// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_inter_retry

import (
	"github.com/rookie-ninja/rk-interceptor/context"
	"github.com/rookie-ninja/rk-query"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"sync"
)

func StreamClientInterceptor(opt ...RetryCallOption) grpc.StreamClientInterceptor {
	initialRetryOpt := mergeOption(defaultOption, opt)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// validate context, if parent context is from logging, then
		// Event in rk_query would exists
		var event rk_query.Event
		if rk_inter_context.IsRkContext(ctx) {
			event = rk_inter_context.GetEvent(ctx)
		}

		// we will check whether Call option contains extra RetryCallOption
		gRpcOpts, newRetryOpts := splitCallOptions(opts)
		retryOpt := mergeOption(initialRetryOpt, newRetryOpts)

		event.SetCounter("rk_max_retries", int64(retryOpt.maxRetries))

		if retryOpt.maxRetries == 0 {
			return streamer(ctx, desc, cc, method, gRpcOpts...)
		}

		if desc.ClientStreams {
			return nil, status.Errorf(codes.Unimplemented, "rk_grpc_retry: cannot retry on ClientStreams, set rk_grpc_retry.Disable()")
		}

		var lastErr error
		for attempt := uint(0); attempt < retryOpt.maxRetries; attempt++ {
			if attempt > 0 {
				event.InCCounter("rk_retry_count", 1)
			}

			if err := waitRetryBackoff(attempt, ctx, event, retryOpt); err != nil {
				return nil, err
			}
			newCtx := callContext(ctx, retryOpt, 0)

			var newStreamer grpc.ClientStream
			newStreamer, lastErr = streamer(newCtx, desc, cc, method, gRpcOpts...)
			if lastErr == nil {
				retryStream := &RetryStream{
					event:        event,
					ClientStream: newStreamer,
					retryOpt:     retryOpt,
					ctx:          ctx,
					streamerCall: func(ctx context.Context) (grpc.ClientStream, error) {
						return streamer(ctx, desc, cc, method, gRpcOpts...)
					},
				}
				return retryStream, nil
			}

			logTrace(ctx, "rk_grpc_retry attempt: %d, got err: %v", attempt, lastErr)
			if isContextError(lastErr) {
				if ctx.Err() != nil {
					logTrace(ctx, "rk_grpc_retry attempt: %d, context error: %v", attempt, ctx.Err())
					// its the parent context deadline or cancellation.
					return nil, lastErr
				} else {
					logTrace(ctx, "rk_grpc_retry attempt: %d, error from retry call", attempt)
					// its the callCtx deadline or cancellation, in which case try again.
					continue
				}
			}
			if !isRetriable(lastErr, retryOpt) {
				return nil, lastErr
			}
		}
		return nil, lastErr
	}
}

type RetryStream struct {
	grpc.ClientStream
	event         rk_query.Event
	bufferedSends []interface{} // single message that the client can sen
	receivedGood  bool          // indicates whether any prior receives were successful
	wasClosedSend bool          // indicates that CloseSend was closed
	ctx           context.Context
	retryOpt      *retryOption
	streamerCall  func(ctx context.Context) (grpc.ClientStream, error)
	mu            sync.RWMutex
}

func (s *RetryStream) setStream(clientStream grpc.ClientStream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ClientStream = clientStream
}

func (s *RetryStream) getStream() grpc.ClientStream {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClientStream
}

func (s *RetryStream) SendMsg(m interface{}) error {
	s.mu.Lock()
	s.bufferedSends = append(s.bufferedSends, m)
	s.mu.Unlock()
	return s.getStream().SendMsg(m)
}

func (s *RetryStream) CloseSend() error {
	s.mu.Lock()
	s.wasClosedSend = true
	s.mu.Unlock()
	return s.getStream().CloseSend()
}

func (s *RetryStream) Header() (metadata.MD, error) {
	return s.getStream().Header()
}

func (s *RetryStream) Trailer() metadata.MD {
	return s.getStream().Trailer()
}

func (s *RetryStream) RecvMsg(m interface{}) error {
	attemptRetry, lastErr := s.receiveMsgAndIndicateRetry(m)
	if !attemptRetry {
		return lastErr // success or hard failure
	}
	// We start off from attempt 1, because zeroth was already made on normal SendMsg().
	for attempt := uint(1); attempt < s.retryOpt.maxRetries; attempt++ {
		s.event.InCCounter("rk_retry_count_recv_msg", 1)

		if err := waitRetryBackoff(attempt, s.ctx, s.event, s.retryOpt); err != nil {
			return err
		}

		callCtx := callContext(s.ctx, s.retryOpt, attempt)
		newStream, err := s.reestablishStreamAndResendBuffer(callCtx)
		if err != nil {
			return err
		}
		s.setStream(newStream)
		attemptRetry, lastErr = s.receiveMsgAndIndicateRetry(m)
		//fmt.Printf("Received message and indicate: %v  %v\n", attemptRetry, lastErr)
		if !attemptRetry {
			return lastErr
		}
	}
	return lastErr
}

func (s *RetryStream) receiveMsgAndIndicateRetry(m interface{}) (bool, error) {
	s.mu.RLock()
	wasGood := s.receivedGood
	s.mu.RUnlock()
	err := s.getStream().RecvMsg(m)
	if err == nil || err == io.EOF {
		s.mu.Lock()
		s.receivedGood = true
		s.mu.Unlock()
		return false, err
	} else if wasGood {
		// previous RecvMsg in the stream succeeded, no retry logic should interfere
		return false, err
	}
	if isContextError(err) {
		if s.ctx.Err() != nil {
			logTrace(s.ctx, "rk_grpc_retry context error: %v", s.ctx.Err())
			return false, err
		} else {
			logTrace(s.ctx, "rk_grpc_retry error from retry call")
			// its the callCtx deadline or cancellation, in which case try again.
			return true, err
		}
	}
	return isRetriable(err, s.retryOpt), err
}

func (s *RetryStream) reestablishStreamAndResendBuffer(callCtx context.Context) (grpc.ClientStream, error) {
	s.mu.RLock()
	bufferedSends := s.bufferedSends
	s.mu.RUnlock()
	newStream, err := s.streamerCall(callCtx)
	if err != nil {
		logTrace(callCtx, "grpc_retry failed redialing new stream: %v", err)
		return nil, err
	}
	for _, msg := range bufferedSends {
		if err := newStream.SendMsg(msg); err != nil {
			logTrace(callCtx, "grpc_retry failed resending message: %v", err)
			return nil, err
		}
	}
	if err := newStream.CloseSend(); err != nil {
		logTrace(callCtx, "grpc_retry failed CloseSend on new stream %v", err)
		return nil, err
	}
	return newStream, nil
}
