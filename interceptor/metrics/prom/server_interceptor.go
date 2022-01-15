// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcmetrics

import (
	"context"
	rkmid "github.com/rookie-ninja/rk-entry/middleware"
	rkmidmetrics "github.com/rookie-ninja/rk-entry/middleware/metrics"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor Create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidmetrics.Option) grpc.UnaryServerInterceptor {
	set := rkmidmetrics.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcMethodKey, info.FullMethod)

		beforeCtx := set.BeforeCtx(nil)

		grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, _, _ := rkgrpcinter.GetGwInfo(rkgrpcctx.GetIncomingHeaders(ctx))
		beforeCtx.Input.GrpcService = grpcService
		beforeCtx.Input.GrpcMethod = grpcMethod
		beforeCtx.Input.GrpcType = "UnaryServer"
		beforeCtx.Input.RestPath = gwPath
		beforeCtx.Input.RestMethod = gwMethod

		set.Before(beforeCtx)

		resp, err := handler(ctx, req)

		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.GrpcErrorKey, err)

		afterCtx := set.AfterCtx(status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return resp, err
	}
}

// StreamServerInterceptor Create new stream server interceptor.
func StreamServerInterceptor(opts ...rkmidmetrics.Option) grpc.StreamServerInterceptor {
	set := rkmidmetrics.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcMethodKey, info.FullMethod)

		beforeCtx := set.BeforeCtx(nil)

		grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, _, _ := rkgrpcinter.GetGwInfo(rkgrpcctx.GetIncomingHeaders(wrappedStream.WrappedContext))
		beforeCtx.Input.GrpcService = grpcService
		beforeCtx.Input.GrpcMethod = grpcMethod
		beforeCtx.Input.GrpcType = "StreamServer"
		beforeCtx.Input.RestPath = gwPath
		beforeCtx.Input.RestMethod = gwMethod

		set.Before(beforeCtx)

		err := handler(srv, wrappedStream)

		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.GrpcErrorKey, err)

		afterCtx := set.AfterCtx(status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return err
	}
}
