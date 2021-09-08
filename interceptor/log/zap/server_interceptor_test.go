// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpclog

import (
	"context"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
)

var (
	fakeRequest  = &FakeRequest{}
	fakeResponse = &FakeResponse{}
	fakeServer   = &FakeServer{}
)

type FakeRequest struct{}

type FakeResponse struct{}

type FakeServer struct{}

type FakeServerStream struct {
	ctx context.Context
}

func (f FakeServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (f FakeServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (f FakeServerStream) SetTrailer(md metadata.MD) {
	return
}

func (f FakeServerStream) Context() context.Context {
	return f.ctx
}

func (f FakeServerStream) SendMsg(m interface{}) error {
	return nil
}

func (f FakeServerStream) RecvMsg(m interface{}) error {
	return nil
}

func TestUnaryServerInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeUnaryServer)])
}

func TestUnaryServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithZapLoggerEntry(rkentry.NoopZapLoggerEntry()),
		WithEventLoggerEntry(rkentry.NoopEventLoggerEntry()))

	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	resp := FakeResponse{}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return resp, nil
	}

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{}))

	res, err := inter(ctx, fakeRequest, info, handler)

	assert.Equal(t, resp, res)
	assert.Nil(t, err)
}

func TestStreamServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithZapLoggerEntry(rkentry.NoopZapLoggerEntry()),
		WithEventLoggerEntry(rkentry.NoopEventLoggerEntry()))

	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{})),
	}, info, handler)

	assert.Nil(t, err)
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
