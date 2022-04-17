// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpclimit

import (
	"context"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/ratelimit"
	"github.com/rookie-ninja/rk-grpc/v2/boot/error"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor Add rate limit interceptors.
func UnaryServerInterceptor(opts ...rkmidlimit.Option) grpc.UnaryServerInterceptor {
	set := rkmidlimit.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcmid.WrapContextForServer(ctx)
		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		set.Before(beforeCtx)

		if beforeCtx.Output.ErrResp != nil {
			return nil, rkgrpcerr.ResourceExhausted(beforeCtx.Output.ErrResp.Err.Message).Err()
		}

		resp, err := handler(ctx, req)
		return resp, err
	}
}

// StreamServerInterceptor Add rate limit interceptors.
func StreamServerInterceptor(opts ...rkmidlimit.Option) grpc.StreamServerInterceptor {
	set := rkmidlimit.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcmid.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		set.Before(beforeCtx)

		if beforeCtx.Output.ErrResp != nil {
			return rkgrpcerr.ResourceExhausted("", beforeCtx.Output.ErrResp.Err).Err()
		}

		return handler(srv, wrappedStream)
	}
}
