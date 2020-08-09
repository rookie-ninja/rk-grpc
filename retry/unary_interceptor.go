// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_inter_retry

import (
	"context"
	"github.com/rookie-ninja/rk-interceptor/context"
	"github.com/rookie-ninja/rk-query"
	"google.golang.org/grpc"
)

func UnaryClientInterceptor(opt ...RetryCallOption) grpc.UnaryClientInterceptor {
	initialRetryOpt := mergeOption(defaultOption, opt)

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
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
			return invoker(ctx, method, req, reply, cc, gRpcOpts...)
		}

		var prevErr error

		for attempt := uint(0); attempt < retryOpt.maxRetries; attempt++ {
			if attempt > 0 {
				event.InCCounter("rk_retry_count", 1)
			}

			// wait for backoff time
			if err := waitRetryBackoff(attempt, ctx, event, retryOpt); err != nil {
				event.AddErr(err)
				return err
			}

			// add metadata and option
			newCtx := callContext(ctx, retryOpt, attempt)

			// call server
			// if no error was detected, then just finish the call
			prevErr = invoker(newCtx, method, req, reply, cc, gRpcOpts...)
			if prevErr == nil {
				event.AddErr(prevErr)
				return nil
			}

			// log to trace
			logTrace(ctx, "rk_retry_count: %d, got err: %v", attempt, prevErr)
			if isContextError(prevErr) {
				if ctx.Err() != nil {
					logTrace(ctx, "rk_retry_count: %d, context error: %v", attempt, ctx.Err())
					return prevErr
				} else {
					logTrace(ctx, "rk_retry_count: %d, error from retry call", attempt)
					continue
				}
			}
			if !isRetriable(prevErr, retryOpt) {
				return prevErr
			}
		}
		return prevErr
	}
}
