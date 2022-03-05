// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/log"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor Create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidlog.Option) grpc.UnaryServerInterceptor {
	set := rkmidlog.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcmid.WrapContextForServer(ctx)
		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		// call before
		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		// remote address
		remoteIp, remotePort, _ := rkgrpcmid.GetRemoteAddressSet(ctx)
		beforeCtx.Input.RemoteAddr = remoteIp + ":" + remotePort

		// grpc and grpc-gateway fields
		grpcService, grpcMethod := rkgrpcmid.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, gwScheme, gwUserAgent := rkgrpcmid.GetGwInfo(rkgrpcctx.GetIncomingHeaders(ctx))
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

		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EventKey, beforeCtx.Output.Event)
		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.LoggerKey, beforeCtx.Output.Logger)

		// call user handler
		resp, err := handler(ctx, req)

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
		wrappedStream.WrappedContext = rkgrpcmid.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		// call before
		beforeCtx := set.BeforeCtx(nil)
		beforeCtx.Input.UrlPath = info.FullMethod

		// remote address
		remoteIp, remotePort, _ := rkgrpcmid.GetRemoteAddressSet(wrappedStream.WrappedContext)
		beforeCtx.Input.RemoteAddr = remoteIp + ":" + remotePort

		// grpc and grpc-gateway fields
		grpcService, grpcMethod := rkgrpcmid.GetGrpcInfo(info.FullMethod)
		gwMethod, gwPath, gwScheme, gwUserAgent := rkgrpcmid.GetGwInfo(rkgrpcctx.GetIncomingHeaders(wrappedStream.WrappedContext))
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

		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EventKey, beforeCtx.Output.Event)
		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.LoggerKey, beforeCtx.Output.Logger)

		// call user handler
		err := handler(srv, wrappedStream)

		// call after
		afterCtx := set.AfterCtx(
			rkgrpcctx.GetRequestId(wrappedStream.WrappedContext),
			rkgrpcctx.GetTraceId(wrappedStream.WrappedContext),
			status.Code(err).String())
		set.After(beforeCtx, afterCtx)

		return err
	}
}
