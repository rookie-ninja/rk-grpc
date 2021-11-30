// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcjwt

import (
	"context"
	"github.com/rookie-ninja/rk-common/error"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor create new unary server interceptor.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcMethodKey, info.FullMethod)

		var err error
		// Before invoking
		if ctx, err = serverBefore(ctx, set, info.FullMethod); err != nil {
			return nil, err
		}

		// Invoking
		return handler(ctx, req)
	}
}

// StreamServerInterceptor create new stream server interceptor.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// Before invoking
		if ctx, err := serverBefore(wrappedStream.WrappedContext, set, info.FullMethod); err != nil {
			return err
		} else {
			wrappedStream.WrappedContext = ctx
		}

		// Invoking
		return handler(srv, wrappedStream)
	}
}

func serverBefore(ctx context.Context, set *optionSet, method string) (context.Context, error) {
	if set.Skipper(method) {
		return ctx, nil
	}

	// extract token from extractor
	var auth string
	var err error
	for _, extractor := range set.extractors {
		// Extract token from extractor, if it's not fail break the loop and
		// set auth
		auth, err = extractor(ctx)
		if err == nil {
			break
		}
	}

	if err != nil {
		return ctx, err
		//return ctx, rkerror.Unauthenticated("invalid or expired jwt", err).Err()
	}

	// parse token
	token, err := set.ParseTokenFunc(auth, ctx)

	if err != nil {
		return ctx, rkerror.Unauthenticated("invalid or expired jwt", err).Err()
	}

	// insert into context
	ctx = context.WithValue(ctx, rkgrpcinter.RpcJwtTokenKey, token)

	return ctx, nil
}
