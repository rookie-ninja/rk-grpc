// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcjwt

import (
	"context"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/jwt"
	"github.com/rookie-ninja/rk-grpc/v2/boot/error"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/context"
	"google.golang.org/grpc"
	"net/http"
	"net/url"
)

// UnaryServerInterceptor create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidjwt.Option) grpc.UnaryServerInterceptor {
	set := rkmidjwt.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcmid.WrapContextForServer(ctx)
		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(createReqByCopyingHeader(ctx, info.FullMethod), nil)
		set.Before(beforeCtx)

		// case 1: error response
		if beforeCtx.Output.ErrResp != nil {
			return nil, rkgrpcerr.Unauthenticated(beforeCtx.Output.ErrResp.Err.Message).Err()
		}

		// insert into context
		ctx = context.WithValue(ctx, rkmid.JwtTokenKey, beforeCtx.Output.JwtToken)

		// case 2: call next
		return handler(ctx, req)
	}
}

// StreamServerInterceptor create new stream server interceptor.
func StreamServerInterceptor(opts ...rkmidjwt.Option) grpc.StreamServerInterceptor {
	set := rkmidjwt.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcmid.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(createReqByCopyingHeader(wrappedStream.WrappedContext, info.FullMethod), nil)
		set.Before(beforeCtx)

		// case 1: error response
		if beforeCtx.Output.ErrResp != nil {
			return rkgrpcerr.Unauthenticated(beforeCtx.Output.ErrResp.Err.Message).Err()
		}

		// insert into context
		wrappedStream.WrappedContext = context.WithValue(wrappedStream.WrappedContext, rkmid.JwtTokenKey, beforeCtx.Output.JwtToken)

		// Invoking
		return handler(srv, wrappedStream)
	}
}

func createReqByCopyingHeader(ctx context.Context, method string) *http.Request {
	req := &http.Request{
		URL: &url.URL{
			Path: method,
		},
		Header: http.Header{},
	}

	for k, list := range rkgrpcctx.GetIncomingHeaders(ctx) {
		if len(list) > 0 {
			req.Header.Set(k, list[0])
		}
	}

	return req
}
