// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"time"
)

// Create new unary server interceptor.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		// Before invoking
		ctx = serverBefore(ctx, set, info.FullMethod, rkgrpcinter.RpcTypeUnaryServer)

		// Invoking
		resp, err := handler(ctx, req)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

		// After invoking
		serverAfter(ctx, set, err)

		return resp, err
	}
}

// Create new stream server interceptor.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		wrappedStream.WrappedContext = serverBefore(wrappedStream.WrappedContext, set, info.FullMethod, rkgrpcinter.RpcTypeStreamServer)

		// Invoking
		err := handler(srv, wrappedStream)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcErrorKey, err)

		// After invoking
		serverAfter(wrappedStream.WrappedContext, set, err)

		return err
	}
}

// Handle logic before handle requests.
func serverBefore(ctx context.Context, set *optionSet, method, grpcType string) context.Context {
	event := set.eventLoggerEntry.GetEventFactory().CreateEvent(
		rkquery.WithZapLogger(set.eventLoggerOverride),
		rkquery.WithEncoding(set.eventLoggerEncoding),
		rkquery.WithAppName(rkentry.GlobalAppCtx.GetAppInfoEntry().AppName),
		rkquery.WithAppVersion(rkentry.GlobalAppCtx.GetAppInfoEntry().Version),
		rkquery.WithEntryName(set.EntryName),
		rkquery.WithEntryType(set.EntryType))

	event.SetStartTime(time.Now())

	incomingHeaders := rkgrpcctx.GetIncomingHeaders(ctx)

	remoteIp, remotePort, _ := rkgrpcinter.GetRemoteAddressSet(ctx)
	grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(method)
	gwMethod, gwPath, gwScheme, gwUserAgent := rkgrpcinter.GetGwInfo(incomingHeaders)

	payloads := []zap.Field{
		zap.String("grpcService", grpcService),
		zap.String("grpcMethod", grpcMethod),
		zap.String("grpcType", grpcType),
		zap.String("gwMethod", gwMethod),
		zap.String("gwPath", gwPath),
		zap.String("gwScheme", gwScheme),
		zap.String("gwUserAgent", gwUserAgent),
	}

	// handle payloads
	event.AddPayloads(payloads...)

	// handle remote address
	event.SetRemoteAddr(remoteIp + ":" + remotePort)

	// handle operation
	event.SetOperation(method)

	if _, ok := ctx.Deadline(); ok {
		event.AddErr(ctx.Err())
	}

	// insert logger and event
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEventKey, event)
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcLoggerKey, set.ZapLogger)

	return ctx
}

// Handle logic after handle requests.
func serverAfter(ctx context.Context, options *optionSet, err error) {
	event := rkgrpcctx.GetEvent(ctx)
	event.AddErr(err)
	code := status.Code(err)
	endTime := time.Now()

	// check whether context is cancelled from client
	select {
	case <-ctx.Done():
		event.AddErr(ctx.Err())
	default:
		break
	}

	if requestId := rkgrpcctx.GetRequestId(ctx); len(requestId) > 0 {
		event.SetRequestId(requestId)
		event.SetEventId(requestId)
	}

	if traceId := rkgrpcctx.GetTraceId(ctx); len(traceId) > 0 {
		event.SetTraceId(traceId)
	}
	event.SetResCode(code.String())
	event.SetEndTime(endTime)
	event.Finish()
}
