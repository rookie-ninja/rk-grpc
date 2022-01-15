// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/middleware"
	"github.com/rookie-ninja/rk-entry/middleware/log"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor Create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidlog.Option) grpc.UnaryServerInterceptor {
	set := rkmidlog.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		// call before
		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		// remote address
		remoteIp, remotePort, _ := rkgrpcinter.GetRemoteAddressSet(ctx)
		beforeCtx.Input.RemoteAddr = remoteIp + ":" + remotePort

		// grpc and grpc-gateway fields
		grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, gwScheme, gwUserAgent := rkgrpcinter.GetGwInfo(rkgrpcctx.GetIncomingHeaders(ctx))
		beforeCtx.Input.Fields = append(beforeCtx.Input.Fields, []zap.Field{
			zap.String("grpcService", grpcService),
			zap.String("grpcMethod", grpcMethod),
			zap.String("grpcType", "UnaryServer"),
			zap.String("gwMethod", gwMethod),
			zap.String("gwPath", gwPath),
			zap.String("gwScheme", gwScheme),
			zap.String("gwUserAgent", gwUserAgent),
		}...)

		set.Before(beforeCtx)

		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EventKey, beforeCtx.Output.Event)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.LoggerKey, beforeCtx.Output.Logger)

		// call user handler
		resp, err := handler(ctx, req)

		// add error if exists
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.GrpcErrorKey, err)

		// call after
		afterCtx := set.AfterCtx(
			rkgrpcctx.GetRequestId(ctx),
			rkgrpcctx.GetTraceId(ctx),
			status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return resp, err
	}
}

// StreamServerInterceptor Create new stream server interceptor.
func StreamServerInterceptor(opts ...rkmidlog.Option) grpc.StreamServerInterceptor {
	set := rkmidlog.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		// call before
		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		// remote address
		remoteIp, remotePort, _ := rkgrpcinter.GetRemoteAddressSet(wrappedStream.WrappedContext)
		beforeCtx.Input.RemoteAddr = remoteIp + ":" + remotePort

		// grpc and grpc-gateway fields
		grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, gwScheme, gwUserAgent := rkgrpcinter.GetGwInfo(rkgrpcctx.GetIncomingHeaders(wrappedStream.WrappedContext))
		beforeCtx.Input.Fields = append(beforeCtx.Input.Fields, []zap.Field{
			zap.String("grpcService", grpcService),
			zap.String("grpcMethod", grpcMethod),
			zap.String("grpcType", "StreamServer"),
			zap.String("gwMethod", gwMethod),
			zap.String("gwPath", gwPath),
			zap.String("gwScheme", gwScheme),
			zap.String("gwUserAgent", gwUserAgent),
		}...)

		set.Before(beforeCtx)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EventKey, beforeCtx.Output.Event)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.LoggerKey, beforeCtx.Output.Logger)

		// call user handler
		err := handler(srv, wrappedStream)

		// add error if exists
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.GrpcErrorKey, err)

		// call after
		afterCtx := set.AfterCtx(
			rkgrpcctx.GetRequestId(wrappedStream.WrappedContext),
			rkgrpcctx.GetTraceId(wrappedStream.WrappedContext),
			status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return err
	}
}
