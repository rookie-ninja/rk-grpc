// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpctrace

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Create new unary server interceptor.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTracerKey, set.Tracer)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTracerProviderKey, set.Provider)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcPropagatorKey, set.Propagator)

		// Before invoking
		ctx, span := serverBefore(ctx, set, info.FullMethod, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcSpanKey, span)

		// Invoking
		resp, err := handler(ctx, req)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

		// After invoking
		serverAfter(span, err)

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
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTracerKey, set.Tracer)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTracerProviderKey, set.Provider)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcPropagatorKey, set.Propagator)

		// Before invoking
		ctx, span := serverBefore(wrappedStream.WrappedContext, set, info.FullMethod, rkgrpcinter.RpcTypeStreamServer)
		wrappedStream.WrappedContext = ctx

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcSpanKey, span)

		// Invoking
		err := handler(srv, wrappedStream)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcErrorKey, err)

		// After invoking
		serverAfter(span, err)

		return err
	}
}

// Convert locale information into attributes.
func localeToAttributes() []attribute.KeyValue {
	res := []attribute.KeyValue{
		attribute.String(rkgrpcinter.Realm.Key, rkgrpcinter.Realm.String),
		attribute.String(rkgrpcinter.Region.Key, rkgrpcinter.Region.String),
		attribute.String(rkgrpcinter.AZ.Key, rkgrpcinter.AZ.String),
		attribute.String(rkgrpcinter.Domain.Key, rkgrpcinter.Domain.String),
	}

	return res
}

// Convert grpc information into attributes.
func grpcInfoToAttributes(ctx context.Context, method, rpcType string) []attribute.KeyValue {
	remoteIp, remotePort, _ := rkgrpcinter.GetRemoteAddressSet(ctx)
	grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(method)
	gwMethod, gwPath, gwScheme, gwUserAgent := rkgrpcinter.GetGwInfo(rkgrpcctx.GetIncomingHeaders(ctx))

	return []attribute.KeyValue{
		attribute.String("local.IP", rkgrpcinter.LocalIp.String),
		attribute.String("local.hostname", rkgrpcinter.LocalHostname.String),
		attribute.String("remote.IP", remoteIp),
		attribute.String("remote.port", remotePort),
		attribute.String("grpc.service", grpcService),
		attribute.String("grpc.method", grpcMethod),
		attribute.String("gw.method", gwMethod),
		attribute.String("gw.path", gwPath),
		attribute.String("gw.scheme", gwScheme),
		attribute.String("gw.userAgent", gwUserAgent),
		attribute.String("server.type", rpcType),
	}
}

// Handle logic before handle requests.
func serverBefore(ctx context.Context, set *optionSet, method, rpcType string) (context.Context, oteltrace.Span) {
	opts := []oteltrace.SpanOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		oteltrace.WithAttributes(localeToAttributes()...),
		oteltrace.WithAttributes(grpcInfoToAttributes(ctx, method, rpcType)...),
	}

	// extract tracer from incoming metadata
	incomingMD, _ := metadata.FromIncomingContext(ctx)
	spanCtx := oteltrace.SpanContextFromContext(set.Propagator.Extract(ctx, &rkgrpcctx.GrpcMetadataCarrier{Md: &incomingMD}))

	// create span name
	spanName := method
	if len(spanName) < 1 {
		spanName = "rk-span-default"
	}

	// create span
	ctx, span := set.Tracer.Start(oteltrace.ContextWithRemoteSpanContext(ctx, spanCtx), spanName, opts...)

	// insert into context
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcctx.TraceIdKey, span.SpanContext().TraceID().String())
	rkgrpcctx.AddHeaderToClient(ctx, rkgrpcctx.TraceIdKey, span.SpanContext().TraceID().String())
	rkgrpcctx.GetEvent(ctx).SetTraceId(span.SpanContext().TraceID().String())

	// return new context with tracer and traceId
	return ctx, span
}

// Handle logic after handle requests.
func serverAfter(span oteltrace.Span, err error) {
	defer span.End()
	if err != nil {
		s, _ := status.FromError(err)
		span.SetStatus(codes.Error, s.Message())
		span.SetAttributes(attribute.Int("grpc.code", int(s.Code())))
		span.SetAttributes(attribute.String("grpc.status", s.Code().String()))
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int("grpc.code", int(codes.Ok)))
		span.SetAttributes(attribute.String("grpc.status", codes.Ok.String()))
	}
}
