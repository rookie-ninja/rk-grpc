// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcctx

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
	"net/http"
	"testing"
)

type FakeClientStream struct {
	ctx context.Context
	md  metadata.MD
}

func (f FakeClientStream) Header() (metadata.MD, error) {
	return f.md, nil
}

func (f FakeClientStream) Trailer() metadata.MD {
	return f.md
}

func (f FakeClientStream) CloseSend() error {
	return nil
}

func (f FakeClientStream) Context() context.Context {
	return f.ctx
}

func (f FakeClientStream) SendMsg(m interface{}) error {
	return nil
}

func (f FakeClientStream) RecvMsg(m interface{}) error {
	return nil
}

func TestGrpcMetadataCarrier_Get(t *testing.T) {
	md := metadata.New(map[string]string{
		"key": "value",
	})
	carrier := &GrpcMetadataCarrier{
		Md: &md,
	}

	// Non exits
	assert.Empty(t, carrier.Get("non-exist"))
	// Exist
	assert.Equal(t, "value", carrier.Get("key"))
}

func TestGrpcMetadataCarrier_Set(t *testing.T) {
	md := metadata.New(map[string]string{})
	carrier := &GrpcMetadataCarrier{
		Md: &md,
	}

	// Non exits
	carrier.Set("key", "value")
	assert.Equal(t, "value", carrier.Get("key"))
}

func TestGrpcMetadataCarrier_Keys(t *testing.T) {
	md := metadata.New(map[string]string{
		"k1": "v1",
		"k2": "v2",
	})
	carrier := &GrpcMetadataCarrier{
		Md: &md,
	}

	// Non exits
	assert.Contains(t, carrier.Keys(), "k1")
	assert.Contains(t, carrier.Keys(), "k2")
}

func TestGetIncomingHeaders(t *testing.T) {
	// On server side
	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.TODO(), md)
	assert.Equal(t, md, GetIncomingHeaders(ctx))

	// Neither of above
	assert.NotNil(t, GetIncomingHeaders(context.TODO()))
}

func TestAddHeaderToClient(t *testing.T) {
	defer assertNotPanic(t)

	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	AddHeaderToClient(ctx, "key", "value")
	assert.Equal(t, "value", rkgrpcmid.GetServerContextPayload(ctx)["key"])
}

func TestGetEvent(t *testing.T) {
	event := rkquery.NewEventFactory().CreateEventNoop()

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EventKey, event)
	assert.Equal(t, event, GetEvent(ctx))

	// For neither of above
	assert.Equal(t, noopEvent, GetEvent(context.TODO()))
}

func TestGetLogger(t *testing.T) {
	logger := rklogger.NoopLogger

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.LoggerKey, logger)
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.HeaderRequestId, "ut-request-id")
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.HeaderTraceId, "ut-trace-id")
	assert.NotNil(t, GetLogger(ctx))

	// For neither of above
	assert.NotNil(t, GetLogger(context.TODO()))
}

func TestGetRequestId(t *testing.T) {
	requestId := "ut-request-id"

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.HeaderRequestId, requestId)
	assert.Equal(t, requestId, GetRequestId(ctx))

	// For neither of above
	assert.Empty(t, GetRequestId(context.TODO()))
}

func TestGetTraceId(t *testing.T) {
	traceId := "ut-trace-id"

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.HeaderTraceId, traceId)
	assert.Equal(t, traceId, GetTraceId(ctx))

	// For neither of above
	assert.Empty(t, GetTraceId(context.TODO()))
}

func TestGetEntryName(t *testing.T) {
	entryName := "ut-entry"

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EntryNameKey, entryName)
	assert.Equal(t, entryName, GetEntryName(ctx))

	// For neither of above
	assert.Empty(t, GetEntryName(context.TODO()))
}

//
//func TestGetRpcType(t *testing.T) {
//	rpcType := "ut-rpc"
//
//	// For server side
//	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
//	rkgrpcmid.AddToServerContextPayload(ctx, rkgrpcmid.GrpcTypeKey, rpcType)
//	assert.Equal(t, rpcType, GetRpcType(ctx))
//
//	// For client side
//	ctx = WrapContext(context.TODO())
//	rkgrpcmid.AddToClientContextPayload(ctx, rkgrpcmid.GrpcTypeKey, rpcType)
//	assert.Equal(t, rpcType, GetRpcType(ctx))
//
//	// For neither of above
//	assert.Empty(t, GetRpcType(context.TODO()))
//}
//
//func TestGetMethodName(t *testing.T) {
//	methodName := "ut-method"
//
//	// For server side
//	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
//	rkgrpcmid.AddToServerContextPayload(ctx, rkgrpcmid.RpcMethodKey, methodName)
//	assert.Equal(t, methodName, GetMethodName(ctx))
//
//	// For client side
//	ctx = WrapContext(context.TODO())
//	rkgrpcmid.AddToClientContextPayload(ctx, rkgrpcmid.RpcMethodKey, methodName)
//	assert.Equal(t, methodName, GetMethodName(ctx))
//
//	// For neither of above
//	assert.Empty(t, GetMethodName(context.TODO()))
//}
//
//func TestGetError(t *testing.T) {
//	err := errors.New("ut-error")
//
//	// For server side
//	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
//	rkgrpcmid.AddToServerContextPayload(ctx, rkgrpcmid.RpcErrorKey, err)
//	assert.Equal(t, err, GetError(ctx))
//
//	// For client side
//	ctx = WrapContext(context.TODO())
//	rkgrpcmid.AddToClientContextPayload(ctx, rkgrpcmid.RpcErrorKey, err)
//	assert.Equal(t, err, GetError(ctx))
//
//	// For neither of above
//	assert.Nil(t, GetError(context.TODO()))
//}

func TestGetTraceSpan(t *testing.T) {
	_, span := noopTracerProvider.Tracer("ut-trace-noop").Start(context.TODO(), "noop-span")

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.SpanKey, span)
	assert.Equal(t, span, GetTraceSpan(ctx))

	// For neither of above
	assert.NotNil(t, GetTraceSpan(context.TODO()))
}

func TestGetTracer(t *testing.T) {
	tracer := noopTracerProvider.Tracer("ut-trace-noop")

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.TracerKey, tracer)
	assert.Equal(t, tracer, GetTracer(ctx))

	// For neither of above
	assert.NotNil(t, GetTracer(context.TODO()))
}

func TestGetTracerProvider(t *testing.T) {
	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.TracerProviderKey, noopTracerProvider)
	assert.Equal(t, noopTracerProvider, GetTracerProvider(ctx))

	// For neither of above
	assert.NotNil(t, GetTracerProvider(context.TODO()))
}

func TestGetTracerPropagator(t *testing.T) {
	prop := propagation.NewCompositeTextMapPropagator()

	// For server side
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.PropagatorKey, prop)
	assert.Equal(t, prop, GetTracerPropagator(ctx))

	// For neither of above
	assert.Nil(t, GetTracerPropagator(context.TODO()))
}

func TestNewTraceSpan(t *testing.T) {
	assert.NotNil(t, NewTraceSpan(context.TODO(), "ut-span"))
}

func TestEndTraceSpan(t *testing.T) {
	defer assertNotPanic(t)

	// For success
	ctx := context.TODO()
	EndTraceSpan(ctx, NewTraceSpan(ctx, "ut-span"), true)

	// For failure
	ctx = context.TODO()
	EndTraceSpan(ctx, NewTraceSpan(ctx, "ut-span"), false)
}

func TestInjectSpanToNewContext(t *testing.T) {
	defer assertNotPanic(t)

	prop := propagation.NewCompositeTextMapPropagator()

	// Inject propagator
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.PropagatorKey, prop)
	assert.Equal(t, prop, GetTracerPropagator(ctx))

	// Inject span
	ctx = InjectSpanToNewContext(ctx)
	_, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
}

func TestInjectSpanToHttpRequest(t *testing.T) {
	defer assertNotPanic(t)

	// For nil request
	InjectSpanToHttpRequest(context.TODO(), nil)

	// For happy case
	prop := propagation.NewCompositeTextMapPropagator()
	// Inject propagator
	ctx := rkgrpcmid.WrapContextForServer(context.TODO())
	rkgrpcmid.AddToServerContextPayload(ctx, rkmid.PropagatorKey, prop)
	assert.Equal(t, prop, GetTracerPropagator(ctx))

	req := &http.Request{
		Header: http.Header{},
	}
	InjectSpanToHttpRequest(ctx, req)
}

func TestGetJwtToken(t *testing.T) {
	// with nil ctx
	assert.Nil(t, GetJwtToken(nil))

	// with invalid type
	ctx := context.WithValue(context.TODO(), rkmid.JwtTokenKey, "token")
	assert.Nil(t, GetJwtToken(ctx))

	// happy case
	token := &jwt.Token{}
	ctx = context.WithValue(context.TODO(), rkmid.JwtTokenKey, token)
	assert.Equal(t, token, GetJwtToken(ctx))
}

func assertNotPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, false)
	} else {
		// This should never be called in case of a bug
		assert.True(t, true)
	}
}
