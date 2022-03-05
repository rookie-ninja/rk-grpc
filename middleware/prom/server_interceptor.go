// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcprom

import (
	"context"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/prom"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor Create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidprom.Option) grpc.UnaryServerInterceptor {
	set := rkmidprom.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcmid.WrapContextForServer(ctx)
		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(nil)

		grpcService, grpcMethod := rkgrpcmid.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, _, _ := rkgrpcmid.GetGwInfo(rkgrpcctx.GetIncomingHeaders(ctx))
		beforeCtx.Input.GrpcService = grpcService
		beforeCtx.Input.GrpcMethod = grpcMethod
		beforeCtx.Input.GrpcType = "UnaryServer"
		beforeCtx.Input.RestPath = gwPath
		beforeCtx.Input.RestMethod = gwMethod

		set.Before(beforeCtx)

		resp, err := handler(ctx, req)

		afterCtx := set.AfterCtx(status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return resp, err
	}
}

// StreamServerInterceptor Create new stream server interceptor.
func StreamServerInterceptor(opts ...rkmidprom.Option) grpc.StreamServerInterceptor {
	set := rkmidprom.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcmid.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(nil)

		grpcService, grpcMethod := rkgrpcmid.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, _, _ := rkgrpcmid.GetGwInfo(rkgrpcctx.GetIncomingHeaders(wrappedStream.WrappedContext))
		beforeCtx.Input.GrpcService = grpcService
		beforeCtx.Input.GrpcMethod = grpcMethod
		beforeCtx.Input.GrpcType = "StreamServer"
		beforeCtx.Input.RestPath = gwPath
		beforeCtx.Input.RestMethod = gwMethod

		set.Before(beforeCtx)

		err := handler(srv, wrappedStream)

		//rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcmid.GrpcErrorKey, err)

		afterCtx := set.AfterCtx(status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return err
	}
}
