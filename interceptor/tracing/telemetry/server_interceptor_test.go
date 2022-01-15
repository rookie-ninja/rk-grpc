// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpctrace

import (
	"context"
	"errors"
	rkmidtrace "github.com/rookie-ninja/rk-entry/middleware/tracing"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor(t *testing.T) {
	//defer assertNotPanic(t)

	beforeCtx := rkmidtrace.NewBeforeCtx()
	afterCtx := rkmidtrace.NewAfterCtx()
	mock := rkmidtrace.NewOptionSetMock(beforeCtx, afterCtx, nil, nil, nil)
	inter := UnaryServerInterceptor(rkmidtrace.WithMockOptionSet(mock))

	// case 1: with error response caused by nil span
	inter(NewUnaryServerInput(true))

	// case 2: happy case
	noopTracerProvider := trace.NewNoopTracerProvider()
	newCtx, span := noopTracerProvider.Tracer("rk-trace-noop").Start(context.TODO(), "noop-span")
	beforeCtx.Output.Span = span
	beforeCtx.Output.NewCtx = newCtx

	inter(NewUnaryServerInput(false))
}

func TestStreamServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	beforeCtx := rkmidtrace.NewBeforeCtx()
	afterCtx := rkmidtrace.NewAfterCtx()
	mock := rkmidtrace.NewOptionSetMock(beforeCtx, afterCtx, nil, nil, nil)
	inter := StreamServerInterceptor(rkmidtrace.WithMockOptionSet(mock))

	// case 1: with error response caused by nil span
	inter(NewStreamServerInput(true))

	// case 2: happy case
	noopTracerProvider := trace.NewNoopTracerProvider()
	newCtx, span := noopTracerProvider.Tracer("rk-trace-noop").Start(context.TODO(), "noop-span")
	beforeCtx.Output.Span = span
	beforeCtx.Output.NewCtx = newCtx

	inter(NewStreamServerInput(false))
}

// ************ Test utility ************

type ServerStreamMock struct {
	ctx context.Context
}

func (f ServerStreamMock) SetHeader(md metadata.MD) error {
	return nil
}

func (f ServerStreamMock) SendHeader(md metadata.MD) error {
	return nil
}

func (f ServerStreamMock) SetTrailer(md metadata.MD) {
	return
}

func (f ServerStreamMock) Context() context.Context {
	return f.ctx
}

func (f ServerStreamMock) SendMsg(m interface{}) error {
	return nil
}

func (f ServerStreamMock) RecvMsg(m interface{}) error {
	return nil
}

func NewUnaryServerInput(withError bool) (context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) {
	ctx := context.TODO()
	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	var handler grpc.UnaryHandler

	if withError {
		handler = func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, errors.New("ut error")
		}
	} else {
		handler = func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, nil
		}
	}

	return ctx, nil, info, handler
}

func NewStreamServerInput(withError bool) (interface{}, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) {
	serverStream := &ServerStreamMock{ctx: context.TODO()}
	info := &grpc.StreamServerInfo{
		FullMethod: "ut-method",
	}

	var handler grpc.StreamHandler
	if withError {
		handler = func(srv interface{}, stream grpc.ServerStream) error {
			return errors.New("ut error")
		}
	} else {
		handler = func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}
	}

	return nil, serverStream, info, handler
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
