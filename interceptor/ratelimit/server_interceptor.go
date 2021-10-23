// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpclimit

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor Add rate limit interceptors.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		event := rkgrpcctx.GetEvent(ctx)

		if duration, err := set.Wait(ctx, info.FullMethod); err != nil {
			event.SetCounter("rateLimitWaitMs", duration.Milliseconds())
			event.AddErr(err)
			return nil, err
		}

		resp, err := handler(ctx, req)
		return resp, err
	}
}

// StreamServerInterceptor Add rate limit interceptors.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		event := rkgrpcctx.GetEvent(wrappedStream.Context())

		if duration, err := set.Wait(wrappedStream.Context(), info.FullMethod); err != nil {
			event.SetCounter("rateLimitWaitMs", duration.Milliseconds())
			event.AddErr(err)
			return err
		}

		return handler(srv, wrappedStream)
	}
}
