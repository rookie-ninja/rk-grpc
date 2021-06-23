// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"time"
)

// Create new unary server interceptor.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// Before invoking
		startTime := time.Now()

		// Invoking
		resp, err := handler(ctx, req)

		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

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

// Create new stream server interceptor.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// 1: Before invoking
		startTime := time.Now()

		// 2: Invoking
		err := handler(srv, wrappedStream)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcErrorKey, err)

		// 3: After invoking
		elapsed := time.Now().Sub(startTime)

		if durationMetrics := GetDurationMetrics(wrappedStream.WrappedContext); durationMetrics != nil {
			durationMetrics.Observe(float64(elapsed.Nanoseconds()))
		}

		if errorMetrics := GetErrorMetrics(wrappedStream.WrappedContext); errorMetrics != nil {
			errorMetrics.Inc()
		}

		if resCodeMetrics := GetResCodeMetrics(wrappedStream.WrappedContext); resCodeMetrics != nil {
			resCodeMetrics.Inc()
		}

		return err
	}
}
