// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_inter_retry

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"time"
)

const (
	RetryCountKey = "x-retry-count"
)

func logTrace(ctx context.Context, format string, a ...interface{}) {
	tr, ok := trace.FromContext(ctx)
	if !ok {
		return
	}
	tr.LazyPrintf(format, a...)
}

func waitRetryBackoff(attempt uint, ctx context.Context, event rk_query.Event, retryOpt *retryOption) error {
	var waitTime time.Duration = 0
	if attempt > 0 {
		waitTime = retryOpt.backoffFunc(attempt)
	}
	event.AddFields(zap.Duration("rk_retry_wait_ms", waitTime))

	if waitTime > 0 {
		logTrace(ctx, "rk_grpc_retry attempt: %d, backoff for %v", attempt, waitTime)
		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			return toGrpcErr(ctx.Err())
		case <-timer.C:
		}
	}
	return nil
}

func isRetriable(err error, retryOpt *retryOption) bool {
	errCode := status.Code(err)
	if isContextError(err) {
		return false
	}
	for _, code := range retryOpt.codes {
		if code == errCode {
			return true
		}
	}
	return false
}

func isContextError(err error) bool {
	return status.Code(err) == codes.DeadlineExceeded || status.Code(err) == codes.Canceled
}

func callContext(ctx context.Context, retryOpt *retryOption, attempt uint) context.Context {
	if retryOpt.callTimeoutMS != 0 {
		ctx, _ = context.WithTimeout(ctx, retryOpt.callTimeoutMS)
	}

	return metadata.AppendToOutgoingContext(ctx, RetryCountKey, fmt.Sprintf("%d", attempt))
}

func toGrpcErr(err error) error {
	switch err {
	case context.DeadlineExceeded:
		return status.Errorf(codes.DeadlineExceeded, err.Error())
	case context.Canceled:
		return status.Errorf(codes.Canceled, err.Error())
	default:
		return status.Errorf(codes.Unknown, err.Error())
	}
}
