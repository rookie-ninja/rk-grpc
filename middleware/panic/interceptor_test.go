// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcpanic

import (
	"context"
	"errors"
	"github.com/rookie-ninja/rk-entry/v2/middleware/panic"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor(t *testing.T) {
	inter := UnaryServerInterceptor(
		rkmidpanic.WithEntryNameAndType("ut-entry", "ut-type"))

	_, err := inter(NewUnaryServerInput())
	assert.Nil(t, err)
}

func TestUnaryServerInterceptor_WithPanic(t *testing.T) {
	inter := UnaryServerInterceptor(
		rkmidpanic.WithEntryNameAndType("ut-entry", "ut-type"))

	ctx, req, info, _ := NewUnaryServerInput()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic(errors.New("ut expected"))
	}

	_, err := inter(ctx, req, info, handler)
	assert.NotNil(t, err)
}

func TestStreamServerInterceptor(t *testing.T) {
	inter := StreamServerInterceptor(
		rkmidpanic.WithEntryNameAndType("ut-entry", "ut-type"))

	err := inter(NewStreamServerInput())

	assert.Nil(t, err)
}

func TestStreamServerInterceptor_WithPanic(t *testing.T) {
	inter := StreamServerInterceptor(
		rkmidpanic.WithEntryNameAndType("ut-entry", "ut-type"))

	svr, stream, info, _ := NewStreamServerInput()

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		panic(errors.New("ut expected"))
	}

	err := inter(svr, stream, info, handler)
	assert.NotNil(t, err)
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

func NewUnaryServerInput() (context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) {
	ctx := context.TODO()
	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	return ctx, nil, info, handler
}

func NewStreamServerInput() (interface{}, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) {
	serverStream := &ServerStreamMock{ctx: context.TODO()}
	info := &grpc.StreamServerInfo{
		FullMethod: "ut-method",
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	return nil, serverStream, info, handler
}
