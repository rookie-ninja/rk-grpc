// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"time"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	newOptionSet(rkgrpcbasic.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Before invoking
		startTime := time.Now()

		// Invoking
		resp, err := handler(ctx, req)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// After invoking
		elapsed := time.Now().Sub(startTime)

		if durationMetrics := GetDurationMetrics(ctx); durationMetrics != nil {
			durationMetrics.Observe(float64(elapsed.Nanoseconds()))
		}

		if errorMetrics := GetErrorMetrics(ctx); errorMetrics != nil {
			errorMetrics.Inc()
		}

		if resCodeMetrics := GetResCodeMetrics(ctx); resCodeMetrics != nil {
			resCodeMetrics.Inc()
		}

		return resp, err
	}
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	newOptionSet(rkgrpcbasic.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		ctx := wrappedStream.Context()

		// 1: Before invoking
		startTime := time.Now()

		// 2: Invoking
		err := handler(srv, wrappedStream)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// 3: After invoking
		elapsed := time.Now().Sub(startTime)

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
