// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcmeta

import (
	"context"
	"github.com/rookie-ninja/rk-entry/middleware"
	"github.com/rookie-ninja/rk-entry/middleware/meta"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor Add common headers as extension style in http response.
func UnaryServerInterceptor(opts ...rkmidmeta.Option) grpc.UnaryServerInterceptor {
	set := rkmidmeta.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(rkgrpcctx.GetEvent(ctx))
		set.Before(beforeCtx)

		for k, v := range beforeCtx.Output.HeadersToReturn {
			rkgrpcctx.AddHeaderToClient(ctx, k, v)
		}

		resp, err := handler(ctx, req)

		return resp, err

	}
}

// StreamServerInterceptor Add common headers as extension style in http response.
func StreamServerInterceptor(opts ...rkmidmeta.Option) grpc.StreamServerInterceptor {
	set := rkmidmeta.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(rkgrpcctx.GetEvent(wrappedStream.WrappedContext))
		set.Before(beforeCtx)

		for k, v := range beforeCtx.Output.HeadersToReturn {
			rkgrpcctx.AddHeaderToClient(wrappedStream.WrappedContext, k, v)
		}

		return handler(srv, wrappedStream)
	}
}
