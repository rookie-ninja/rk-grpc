// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcctx

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strings"
	"time"
)

const (
	RequestIdKey = "X-Request-Id"
	TraceIdKey   = "X-Trace-Id"
)

var (
	noopTracerProvider = trace.NewNoopTracerProvider()
	noopEvent          = rkquery.NewEventFactory().CreateEventNoop()
)

// Grpc metadata carrier which will carries tracing info into grpc metadata to server side.
type GrpcMetadataCarrier struct {
	Md *metadata.MD
}

// Get value with key from grpc metadata.
func (carrier *GrpcMetadataCarrier) Get(key string) string {
	values := carrier.Md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set value with key into grpc metadata.
func (carrier *GrpcMetadataCarrier) Set(key string, value string) {
	carrier.Md.Set(key, value)
}

// List keys in grpc metadata.
func (carrier *GrpcMetadataCarrier) Keys() []string {
	out := make([]string, 0, len(*carrier.Md))
	for key := range *carrier.Md {
		out = append(out, key)
	}
	return out
}

// We will add payload into context for further usage.
func WrapContext(ctx context.Context) context.Context {
	if rkgrpcinter.ContainsClientPayload(ctx) {
		return ctx
	}

	return context.WithValue(ctx, rkgrpcinter.GetClientPayloadKey(), rkgrpcinter.NewClientPayload())
}

// This function is mainly used for client stream.
//
// Streaming client is a little bit tricky.
// It is not an easy work to get headers sent from server while receiving message
// since client stream interceptor will finish before client start receiving message.
//
// As a result, what event will log is the time before Recv() start to be called.
// No request id nor trace id would be logged since we are unable to call stream.Header() function which would be
// blocked until stream.Recv() has been called.
//
// We believe it is not a good idea to wrap client stream or do anything tricky with stream.
//
// If user hope to log request id and trace id into event, user need to call bellow function as soon as stream.Header()
// is ready.
// The downside is you will get multiple event logs with same event id.
func FinishClientStream(ctx context.Context, stream grpc.ClientStream) {
	if header, err := stream.Header(); err == nil {
		event := GetEvent(ctx)

		requestId := strings.Join(header.Get(RequestIdKey), ",")
		if len(requestId) > 0 {
			event.SetRequestId(requestId)
			rkgrpcinter.AddToClientContextPayload(ctx, RequestIdKey, requestId)
		}

		traceId := strings.Join(header.Get(TraceIdKey), ",")
		if len(traceId) > 0 {
			event.SetTraceId(traceId)
			rkgrpcinter.AddToClientContextPayload(ctx, TraceIdKey, traceId)
		}

		event.SetEndTime(time.Now())
		event.Finish()
	}
}

// Extract call-scoped incoming headers
func GetIncomingHeaders(ctx context.Context) metadata.MD {
	// called from client
	if rkgrpcinter.ContainsClientPayload(ctx) {
		return *rkgrpcinter.GetIncomingHeadersOfClient(ctx)
	}

	// called from server
	if v, ok := metadata.FromIncomingContext(ctx); ok {
		return v
	}

	return metadata.Pairs()
}

// Headers that would be sent to client.
func AddHeaderToClient(ctx context.Context, key, value string) {
	// set to grpc header
	if err := grpc.SetHeader(ctx, metadata.Pairs(key, value)); err != nil {
		GetLogger(ctx).Warn("Failed to write to grpc header at server side", zap.String("key", key))
	}

	rkgrpcinter.AddToServerContextPayload(ctx, key, value)
}

// Headers that would be sent to server.
func AddHeaderToServer(ctx context.Context, key, value string) {
	// Make sure called from client
	if rkgrpcinter.ContainsClientPayload(ctx) {
		// called from client side
		outgoingHeaders := rkgrpcinter.GetOutgoingHeadersOfClient(ctx)
		outgoingHeaders.Append(key, value)
	}
}

// Extract the call-scoped EventData from context.
func GetEvent(ctx context.Context) rkquery.Event {
	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v, ok := m[rkgrpcinter.RpcEventKey]; ok {
		return v.(rkquery.Event)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v, ok := m[rkgrpcinter.RpcEventKey]; ok {
		return v.(rkquery.Event)
	}

	return noopEvent
}

// Extract the call-scoped zap logger from context.
func GetLogger(ctx context.Context) *zap.Logger {
	logger := rklogger.NoopLogger

	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcLoggerKey]; ok {
		requestId := GetRequestId(ctx)
		traceId := GetTraceId(ctx)
		fields := make([]zap.Field, 0)
		if len(requestId) > 0 {
			fields = append(fields, zap.String("requestId", requestId))
		}
		if len(traceId) > 0 {
			fields = append(fields, zap.String("traceId", traceId))
		}
		return v1.(*zap.Logger).With(fields...)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcLoggerKey]; ok {
		requestId := GetRequestId(ctx)
		traceId := GetTraceId(ctx)
		fields := make([]zap.Field, 0)
		if len(requestId) > 0 {
			fields = append(fields, zap.String("requestId", requestId))
		}
		if len(traceId) > 0 {
			fields = append(fields, zap.String("traceId", traceId))
		}
		return v1.(*zap.Logger).With(fields...)
	}

	return logger
}

// Get request id in outgoing metadata.
func GetRequestId(ctx context.Context) string {
	// case 1: called from server side context which wrapped with WrapContextForServer()'
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if id := m[RequestIdKey]; id != nil {
		return id.(string)
	}

	// case 2: called from client side context which wrapped with WrapContextForClient()
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[RequestIdKey]; ok {
		return v1.(string)
	}

	return ""
}

// Get trace id in context.
func GetTraceId(ctx context.Context) string {
	// case 1: called from server side context which wrapped with WrapContextForServer()
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if id := m[TraceIdKey]; id != nil {
		return id.(string)
	}

	// case 2: called from client side context which wrapped with WrapContextForClient()
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[TraceIdKey]; ok {
		return v1.(string)
	}

	return ""
}

// Extract the call-scoped entry name.
func GetEntryName(ctx context.Context) string {
	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcEntryNameKey]; ok {
		return v1.(string)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcEntryNameKey]; ok {
		return v1.(string)
	}

	return ""
}

// Extract the call-scoped rpc type.
func GetRpcType(ctx context.Context) string {
	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcTypeKey]; ok {
		return v1.(string)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcTypeKey]; ok {
		return v1.(string)
	}

	return ""
}

// Extract the call-scoped method name.
func GetMethodName(ctx context.Context) string {
	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcMethodKey]; ok {
		return v1.(string)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcMethodKey]; ok {
		return v1.(string)
	}

	return ""
}

// Extract the call-scoped error.
func GetError(ctx context.Context) error {
	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcErrorKey]; ok {
		return v1.(error)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcErrorKey]; ok {
		return v1.(error)
	}

	return nil
}

// Extract the call-scoped span from context.
func GetTraceSpan(ctx context.Context) trace.Span {
	_, span := noopTracerProvider.Tracer("rk-trace-noop").Start(ctx, "noop-span")

	if ctx == nil {
		return span
	}

	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcSpanKey]; ok {
		return v1.(trace.Span)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcSpanKey]; ok {
		return v1.(trace.Span)
	}

	return span
}

// Extract the call-scoped tracer from context.
func GetTracer(ctx context.Context) trace.Tracer {
	if ctx == nil {
		return noopTracerProvider.Tracer("rk-trace-noop")
	}

	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcTracerKey]; ok {
		return v1.(trace.Tracer)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcTracerKey]; ok {
		return v1.(trace.Tracer)
	}

	return noopTracerProvider.Tracer("rk-trace-noop")
}

// Extract the call-scoped tracer provider from context.
func GetTracerProvider(ctx context.Context) trace.TracerProvider {
	if ctx == nil {
		return noopTracerProvider
	}

	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcTracerProviderKey]; ok {
		return v1.(trace.TracerProvider)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcTracerProviderKey]; ok {
		return v1.(trace.TracerProvider)
	}

	return noopTracerProvider
}

// Extract the call-scoped span processor from middleware.
func GetTracerPropagator(ctx context.Context) propagation.TextMapPropagator {
	if ctx == nil {
		return nil
	}

	// case 1: called from server side
	m := rkgrpcinter.GetServerContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcPropagatorKey]; ok {
		return v1.(propagation.TextMapPropagator)
	}

	// case 2: called from client side
	m = rkgrpcinter.GetClientContextPayload(ctx)
	if v1, ok := m[rkgrpcinter.RpcPropagatorKey]; ok {
		return v1.(propagation.TextMapPropagator)
	}

	return nil
}

// Start a new span
func NewTraceSpan(ctx context.Context, name string) trace.Span {
	tracer := GetTracer(ctx)
	_, span := tracer.Start(ctx, name)

	return span
}

// End span
func EndTraceSpan(ctx context.Context, span trace.Span, success bool) {
	if success {
		span.SetStatus(otelcodes.Ok, otelcodes.Ok.String())
	}

	span.End()
}

// Inject current trace information into context
func InjectSpanToNewContext(ctx context.Context) context.Context {
	newCtx := trace.ContextWithRemoteSpanContext(context.Background(), GetTraceSpan(ctx).SpanContext())
	md := metadata.Pairs()
	GetTracerPropagator(ctx).Inject(newCtx, &GrpcMetadataCarrier{Md: &md})
	newCtx = metadata.NewOutgoingContext(newCtx, md)

	return newCtx
}

// Inject current trace information into http request
func InjectSpanToHttpRequest(ctx context.Context, req *http.Request) {
	if req == nil {
		return
	}

	newCtx := trace.ContextWithRemoteSpanContext(req.Context(), GetTraceSpan(ctx).SpanContext())
	GetTracerPropagator(ctx).Inject(newCtx, propagation.HeaderCarrier(req.Header))
}
