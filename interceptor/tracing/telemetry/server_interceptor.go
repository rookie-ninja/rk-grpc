package rkgrpctrace

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Before invoking
		newCtx, span := serverBefore(ctx, set)

		// Invoking
		resp, err := handler(newCtx, req)

		if rpcInfo := rkgrpcctx.GetRpcInfo(newCtx); rpcInfo != nil {
			rpcInfo.Err = err
		}

		// After invoking
		serverAfter(span, err)

		return resp, err
	}
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		ctx := wrappedStream.WrappedContext

		// Before invoking
		newCtx, span := serverBefore(ctx, set)

		// Invoking
		err := handler(srv, wrappedStream)

		if rpcInfo := rkgrpcctx.GetRpcInfo(newCtx); rpcInfo != nil {
			rpcInfo.Err = err
		}

		// After invoking
		serverAfter(span, err)

		return err
	}
}

func localeToAttributes() []attribute.KeyValue {
	res := []attribute.KeyValue{
		attribute.String(rkgrpcbasic.Realm.Key, rkgrpcbasic.Realm.String),
		attribute.String(rkgrpcbasic.Region.Key, rkgrpcbasic.Region.String),
		attribute.String(rkgrpcbasic.AZ.Key, rkgrpcbasic.AZ.String),
		attribute.String(rkgrpcbasic.Domain.Key, rkgrpcbasic.Domain.String),
	}

	return res
}

func grpcInfoToAttributes(ctx context.Context) []attribute.KeyValue {
	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)

	res := []attribute.KeyValue{
		attribute.String("local.IP", rkgrpcbasic.LocalIp.String),
		attribute.String("local.hostname", rkgrpcbasic.LocalHostname.String),
		attribute.String("remote.IP", rpcInfo.RemoteIp),
		attribute.String("remote.port", rpcInfo.RemotePort),
		attribute.String("grpc.service", rpcInfo.GrpcService),
		attribute.String("grpc.method", rpcInfo.GrpcMethod),
		attribute.String("gw.method", rpcInfo.GwMethod),
		attribute.String("gw.path", rpcInfo.GwPath),
		attribute.String("gw.scheme", rpcInfo.GwScheme),
		attribute.String("gw.userAgent", rpcInfo.GwUserAgent),
		attribute.String("server.type", rpcInfo.Type),
	}

	return res
}

func serverBefore(ctx context.Context, set *optionSet) (context.Context, oteltrace.Span) {
	opts := []oteltrace.SpanOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		oteltrace.WithAttributes(localeToAttributes()...),
		oteltrace.WithAttributes(grpcInfoToAttributes(ctx)...),
	}

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)

	// extract tracer from incoming metadata
	incomingMD, _ := metadata.FromIncomingContext(ctx)
	spanCtx := oteltrace.SpanContextFromContext(set.Propagator.Extract(ctx, &GrpcMetadataCarrier{md: &incomingMD}))

	// create span name
	spanName := rpcInfo.GrpcMethod
	if len(spanName) < 1 {
		spanName = "rk-span-default"
	}

	// create span
	ctx, span := set.Tracer.Start(oteltrace.ContextWithRemoteSpanContext(ctx, spanCtx), spanName, opts...)

	// insert into context
	ctx = context.WithValue(ctx, "rk-trace-id", span.SpanContext().TraceID().String())
	rkgrpcctx.GetEvent(ctx).SetTraceId(span.SpanContext().TraceID().String())

	// return new context with tracer and traceId
	return rkgrpcctx.ToRkContext(ctx,
		rkgrpcctx.WithTracer(set.Tracer),
		rkgrpcctx.WithPropagator(set.Propagator),
		rkgrpcctx.WithTraceProvider(set.Provider)), span
}

func serverAfter(span oteltrace.Span, err error) {
	defer span.End()
	if err != nil {
		s, _ := status.FromError(err)
		span.SetStatus(codes.Error, s.Message())
		span.SetAttributes(attribute.Int("grpc.code", int(s.Code())))
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int("grpc.code", int(codes.Ok)))
	}
}
