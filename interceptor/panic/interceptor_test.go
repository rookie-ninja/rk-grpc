// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcpanic

import (
	"context"
	"errors"
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

func TestUnaryServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

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

func TestUnaryServerInterceptor_WithPanic(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic(errors.New("ut expected"))
	}

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{}))

	res, err := inter(ctx, fakeRequest, info, handler)

	assert.Nil(t, res)
	assert.NotNil(t, err)
}

func TestStreamServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	info := &grpc.StreamServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{})),
	}, info, handler)

	assert.Nil(t, err)
}

func TestStreamServerInterceptor_WithPanic(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	info := &grpc.StreamServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		panic(errors.New("ut expected"))
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{})),
	}, info, handler)

	assert.NotNil(t, err)
}

func TestUnaryClientInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, "ut-method", fakeRequest, fakeResponse, cc, invoker)
}

func TestUnaryClientInterceptor_WithPanic(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		panic(errors.New("ut expected"))
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, "ut-method", fakeRequest, fakeResponse, cc, invoker)
}

func TestStreamClientInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return nil, nil
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, &grpc.StreamDesc{}, cc, "ut-method", streamer)
}

func TestStreamClientInterceptor_WithPanic(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"))

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		panic(errors.New("ut expected"))
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, &grpc.StreamDesc{}, cc, "ut-method", streamer)
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
