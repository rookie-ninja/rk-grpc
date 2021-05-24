// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"time"
)

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	newOptionSet(rkgrpcctx.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Before invoking
		startTime := time.Now()

		// Invoking
		err := invoker(ctx, method, req, resp, cc, opts...)

		elapsed := time.Now().Sub(startTime)

		// After invoking
		if durationMetrics := GetDurationMetrics(ctx); durationMetrics != nil {
			durationMetrics.Observe(float64(elapsed.Nanoseconds()))
		}

		if errorMetrics := GetErrorMetrics(ctx); errorMetrics != nil {
			errorMetrics.Inc()
		}

		if resCodeMetrics := GetResCodeMetrics(ctx); resCodeMetrics != nil {
			resCodeMetrics.Inc()
		}

		return err
	}
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	newOptionSet(rkgrpcctx.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Before invoking
		startTime := time.Now()

		// Invoking
		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		elapsed := time.Now().Sub(startTime)

		// After invoking
		if durationMetrics := GetDurationMetrics(ctx); durationMetrics != nil {
			durationMetrics.Observe(float64(elapsed.Nanoseconds()))
		}

		if errorMetrics := GetErrorMetrics(ctx); errorMetrics != nil {
			errorMetrics.Inc()
		}

		if resCodeMetrics := GetResCodeMetrics(ctx); resCodeMetrics != nil {
			resCodeMetrics.Inc()
		}

		return clientStream, err
	}
}
