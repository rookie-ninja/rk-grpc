// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"google.golang.org/grpc"
	"time"
)

// Create new unary client interceptor.
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryClient)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcMethodKey, method)

		// Before invoking
		startTime := time.Now()

		// Invoking
		err := invoker(ctx, method, req, resp, cc, opts...)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

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

// Create new stream client interceptor.
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeStreamClient)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcMethodKey, method)

		// Before invoking
		startTime := time.Now()
		ctx = context.WithValue(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeStreamClient)

		// Invoking
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

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
