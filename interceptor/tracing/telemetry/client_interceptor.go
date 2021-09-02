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
	"strings"
)

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor suitable
// for use in a grpc.Dial call.
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcTracerKey, set.Tracer)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcTracerProviderKey, set.Provider)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcPropagatorKey, set.Propagator)

		// 1: Before invoking
		ctx, span := clientBefore(ctx, set, method, rkgrpcinter.RpcTypeUnaryClient)

		opts = append(opts, grpc.Header(rkgrpcinter.GetIncomingHeadersOfClient(ctx)))

		// 2: Invoking
		err := invoker(ctx, method, req, resp, cc, opts...)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

		// 3: After invoking
		clientAfter(ctx, span, err)

		return err
	}
}

// Create new stream client interceptor.
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcTracerKey, set.Tracer)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcTracerProviderKey, set.Provider)
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcPropagatorKey, set.Propagator)

		// Before invoking
		ctx, span := clientBefore(ctx, set, method, rkgrpcinter.RpcTypeStreamClient)

		opts = append(opts, grpc.Header(rkgrpcinter.GetIncomingHeadersOfClient(ctx)))

		// Invoking
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcErrorKey, err)

		// After invoking
		clientAfter(ctx, span, err)

		return clientStream, err
	}
}

// Handle logic before handle requests.
func clientBefore(ctx context.Context, set *optionSet, method, rpcType string) (context.Context, oteltrace.Span) {
	opts := []oteltrace.SpanOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(localeToAttributes()...),
		oteltrace.WithAttributes(grpcInfoToAttributes(ctx, method, rpcType)...),
	}

	// create span name
	spanName := method
	if len(spanName) < 1 {
		spanName = "rk-span-default"
	}

	// create span
	ctx, span := set.Tracer.Start(ctx, spanName, opts...)

	// inject into metadata
	md := metadata.Pairs()
	set.Propagator.Inject(ctx, &rkgrpcctx.GrpcMetadataCarrier{Md: &md})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// return new context with tracer and traceId
	return ctx, span
}

// Handle logic after handle requests.
func clientAfter(ctx context.Context, span oteltrace.Span, err error) {
	defer span.End()

	ids := []string{span.SpanContext().TraceID().String()}
	incomingMD := rkgrpcinter.GetIncomingHeadersOfClient(ctx)
	if v := incomingMD.Get(rkgrpcctx.TraceIdKey); len(v) > 0 {
		// We got X-Trace-Id header from server.
		// There are two cases here.
		// case 1:
		// Server read trace info we sent and use same trace id and sent it back.
		// Then, the id would be the same.
		//
		// case 2:
		// Server did not read trace info we sent and use different trace id and sent it back.
		// Then, the id would be different.
		//
		// We will deduplicate outgoing and incoming trace id and set it into client context payload.
		ids = rkgrpcinter.MergeAndDeduplicateSlice(ids, v)
	}

	rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcctx.TraceIdKey, strings.Join(ids, ","))
	rkgrpcctx.GetEvent(ctx).SetTraceId(rkgrpcctx.GetTraceId(ctx))

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
