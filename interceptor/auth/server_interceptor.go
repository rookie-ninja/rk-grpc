// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpcauth is auth interceptor for grpc framework
package rkgrpcauth

import (
	"context"
	"github.com/rookie-ninja/rk-entry/middleware"
	"github.com/rookie-ninja/rk-entry/middleware/auth"
	"github.com/rookie-ninja/rk-grpc/boot/error"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerInterceptor create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidauth.Option) grpc.UnaryServerInterceptor {
	set := rkmidauth.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.GrpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// 1: create beforeCtx
		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		// 2: assign values
		md := rkgrpcctx.GetIncomingHeaders(ctx)

		beforeCtx.Input.BasicAuthHeader = getFirstHeader(md, rkmid.HeaderAuthorization)
		beforeCtx.Input.ApiKeyHeader = getFirstHeader(md, rkmid.HeaderApiKey)

		// 3: call before
		set.Before(beforeCtx)

		// case 1: return to user if error occur
		if beforeCtx.Output.ErrResp != nil {
			for k, v := range beforeCtx.Output.HeadersToReturn {
				rkgrpcctx.AddHeaderToClient(ctx, k, v)
			}

			return nil, rkgrpcerr.Unauthenticated(beforeCtx.Output.ErrResp.Err.Message).Err()
		}

		// case 2: authorized, call next
		return handler(ctx, req)
	}
}

// StreamServerInterceptor create new stream server interceptor.
func StreamServerInterceptor(opts ...rkmidauth.Option) grpc.StreamServerInterceptor {
	set := rkmidauth.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// 1: create beforeCtx
		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		// 2: assign values
		md := rkgrpcctx.GetIncomingHeaders(wrappedStream.WrappedContext)
		beforeCtx.Input.BasicAuthHeader = getFirstHeader(md, rkmid.HeaderAuthorization)
		beforeCtx.Input.ApiKeyHeader = getFirstHeader(md, rkmid.HeaderApiKey)

		// 3: call before
		set.Before(beforeCtx)

		// case 1: return to user if error occur
		if beforeCtx.Output.ErrResp != nil {
			for k, v := range beforeCtx.Output.HeadersToReturn {
				rkgrpcctx.AddHeaderToClient(wrappedStream.WrappedContext, k, v)
			}

			return rkgrpcerr.Unauthenticated(beforeCtx.Output.ErrResp.Err.Message).Err()
		}

		// case 2: authorized, call next
		return handler(srv, wrappedStream)
	}
}

func getFirstHeader(md metadata.MD, key string) string {
	headers := md.Get(key)

	if len(headers) > 0 {
		return headers[0]
	}

	return ""
}
