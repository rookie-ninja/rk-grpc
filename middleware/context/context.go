// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpcctx provides utility functions deal with metadata in RPC context
package rkgrpcctx

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
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
)

var (
	noopTracerProvider = trace.NewNoopTracerProvider()
	noopEvent          = rkquery.NewEventFactory().CreateEventNoop()
)

// GrpcMetadataCarrier Grpc metadata carrier which will carries tracing info into grpc metadata to server side.
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

// Keys List keys in grpc metadata.
func (carrier *GrpcMetadataCarrier) Keys() []string {
	out := make([]string, 0, len(*carrier.Md))
	for key := range *carrier.Md {
		out = append(out, key)
	}
	return out
}

// GetIncomingHeaders Extract call-scoped incoming headers
func GetIncomingHeaders(ctx context.Context) metadata.MD {
	// called from server
	if v, ok := metadata.FromIncomingContext(ctx); ok {
		return v
	}

	return metadata.Pairs()
}

// AddHeaderToClient Headers that would be sent to client.
func AddHeaderToClient(ctx context.Context, key, value string) {
	// set to grpc header
	if err := grpc.SetHeader(ctx, metadata.Pairs(key, value)); err != nil {
		GetLogger(ctx).Warn("Failed to write to grpc header at server side", zap.String("key", key))
	}

	rkgrpcmid.AddToServerContextPayload(ctx, key, value)
}

// GetEvent Extract the call-scoped EventData from context.
func GetEvent(ctx context.Context) rkquery.Event {
	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v, ok := m[rkmid.EventKey]; ok {
		return v.(rkquery.Event)
	}

	return noopEvent
}

// GetLogger Extract the call-scoped zap logger from context.
func GetLogger(ctx context.Context) *zap.Logger {
	logger := rklogger.NoopLogger

	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v1, ok := m[rkmid.LoggerKey]; ok {
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

// GetRequestId Get request id in outgoing metadata.
func GetRequestId(ctx context.Context) string {
	// case 1: called from server side context which wrapped with WrapContextForServer()'
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if id := m[rkmid.HeaderRequestId]; id != nil {
		return id.(string)
	}

	return ""
}

// GetTraceId Get trace id in context.
func GetTraceId(ctx context.Context) string {
	// case 1: called from server side context which wrapped with WrapContextForServer()
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if id := m[rkmid.HeaderTraceId]; id != nil {
		return id.(string)
	}

	return ""
}

// GetEntryName Extract the call-scoped entry name.
func GetEntryName(ctx context.Context) string {
	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v1, ok := m[rkmid.EntryNameKey]; ok {
		return v1.(string)
	}

	return ""
}

// GetTraceSpan Extract the call-scoped span from context.
func GetTraceSpan(ctx context.Context) trace.Span {
	_, span := noopTracerProvider.Tracer("rk-trace-noop").Start(ctx, "noop-span")

	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v1, ok := m[rkmid.SpanKey]; ok {
		return v1.(trace.Span)
	}

	return span
}

// GetTracer Extract the call-scoped tracer from context.
func GetTracer(ctx context.Context) trace.Tracer {
	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v1, ok := m[rkmid.TracerKey]; ok {
		return v1.(trace.Tracer)
	}

	return noopTracerProvider.Tracer("rk-trace-noop")
}

// GetTracerProvider Extract the call-scoped tracer provider from context.
func GetTracerProvider(ctx context.Context) trace.TracerProvider {
	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v1, ok := m[rkmid.TracerProviderKey]; ok {
		return v1.(trace.TracerProvider)
	}

	return noopTracerProvider
}

// GetTracerPropagator Extract the call-scoped span processor from middleware.
func GetTracerPropagator(ctx context.Context) propagation.TextMapPropagator {
	// case 1: called from server side
	m := rkgrpcmid.GetServerContextPayload(ctx)
	if v1, ok := m[rkmid.PropagatorKey]; ok {
		return v1.(propagation.TextMapPropagator)
	}

	return nil
}

// NewTraceSpan Start a new span
func NewTraceSpan(ctx context.Context, name string) trace.Span {
	tracer := GetTracer(ctx)
	_, span := tracer.Start(ctx, name)

	return span
}

// EndTraceSpan End span
func EndTraceSpan(ctx context.Context, span trace.Span, success bool) {
	if success {
		span.SetStatus(otelcodes.Ok, otelcodes.Ok.String())
	}

	span.End()
}

// InjectSpanToNewContext Inject current trace information into context
func InjectSpanToNewContext(ctx context.Context) context.Context {
	newCtx := trace.ContextWithRemoteSpanContext(context.Background(), GetTraceSpan(ctx).SpanContext())
	md := metadata.Pairs()
	GetTracerPropagator(ctx).Inject(newCtx, &GrpcMetadataCarrier{Md: &md})
	newCtx = metadata.NewOutgoingContext(newCtx, md)

	return newCtx
}

// InjectSpanToHttpRequest Inject current trace information into http request
func InjectSpanToHttpRequest(ctx context.Context, req *http.Request) {
	if req == nil {
		return
	}

	newCtx := trace.ContextWithRemoteSpanContext(req.Context(), GetTraceSpan(ctx).SpanContext())
	GetTracerPropagator(ctx).Inject(newCtx, propagation.HeaderCarrier(req.Header))
}

// GetJwtToken return jwt.Token if exists
func GetJwtToken(ctx context.Context) *jwt.Token {
	if ctx == nil {
		return nil
	}

	if raw := ctx.Value(rkmid.JwtTokenKey); raw != nil {
		if res, ok := raw.(*jwt.Token); ok {
			return res
		}
	}

	return nil
}
