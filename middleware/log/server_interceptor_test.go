// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpclog

import (
	"context"
	"github.com/rookie-ninja/rk-entry/v2/entry"
	"github.com/rookie-ninja/rk-entry/v2/middleware/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor(t *testing.T) {
	beforeCtx := rkmidlog.NewBeforeCtx()
	afterCtx := rkmidlog.NewAfterCtx()
	mock := rkmidlog.NewOptionSetMock(beforeCtx, afterCtx)
	inter := UnaryServerInterceptor(rkmidlog.WithMockOptionSet(mock))

	event := rkentry.EventEntryNoop.CreateEventNoop()
	logger := rkentry.LoggerEntryNoop.Logger

	beforeCtx.Output.Event = event
	beforeCtx.Output.Logger = logger
	_, err := inter(NewUnaryServerInput())
	assert.Nil(t, err)
}

func TestStreamServerInterceptor(t *testing.T) {
	beforeCtx := rkmidlog.NewBeforeCtx()
	afterCtx := rkmidlog.NewAfterCtx()
	mock := rkmidlog.NewOptionSetMock(beforeCtx, afterCtx)
	inter := StreamServerInterceptor(rkmidlog.WithMockOptionSet(mock))

	event := rkentry.EventEntryNoop.CreateEventNoop()
	logger := rkentry.LoggerEntryNoop.Logger

	beforeCtx.Output.Event = event
	beforeCtx.Output.Logger = logger
	err := inter(NewStreamServerInput())
	assert.Nil(t, err)
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
