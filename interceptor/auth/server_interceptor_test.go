// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcauth

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func TestUnaryServerInterceptor_WithIgnoringPath(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"),
		WithIgnorePrefix("ut-ignore-path"))

	info := &grpc.UnaryServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}

	res, err := inter(context.TODO(), fakeRequest, info, handler)
	assert.NotNil(t, res)
	assert.Nil(t, err)
}

func TestUnaryServerInterceptor_WithBasicAuth_Invalid(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{
		rkgrpcinter.RpcAuthorizationHeaderKey: "invalid",
	}))

	res, err := inter(ctx, fakeRequest, info, handler)

	assert.Nil(t, res)
	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestUnaryServerInterceptor_InvalidBasicAuth(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{
		rkgrpcinter.RpcAuthorizationHeaderKey: fmt.Sprintf("%s invalid", typeBasic),
	}))

	res, err := inter(ctx, fakeRequest, info, handler)

	assert.Nil(t, res)
	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestUnaryServerInterceptor_WithApiKey_Invalid(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{
		rkgrpcinter.RpcApiKeyHeaderKey: "invalid",
	}))

	res, err := inter(ctx, fakeRequest, info, handler)

	assert.Nil(t, res)
	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestUnaryServerInterceptor_MissingAuth(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.UnaryServerInfo{
		FullMethod: "ut-method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{}))

	res, err := inter(ctx, fakeRequest, info, handler)

	assert.Nil(t, res)
	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestStreamServerInterceptor_WithIgnoringPath(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"),
		WithIgnorePrefix("ut-ignore-path"))

	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: context.TODO(),
	}, info, handler)
	assert.Nil(t, err)
}

func TestStreamServerInterceptor_WithBasicAuth_Invalid(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{
			rkgrpcinter.RpcAuthorizationHeaderKey: "invalid",
		})),
	}, info, handler)

	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestStreamServerInterceptor_InvalidBasicAuth(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{
			rkgrpcinter.RpcAuthorizationHeaderKey: fmt.Sprintf("%s invalid", typeBasic),
		})),
	}, info, handler)

	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestStreamServerInterceptor_WithApiKey_Invalid(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{
			rkgrpcinter.RpcApiKeyHeaderKey: "invalid",
		})),
	}, info, handler)

	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

func TestStreamServerInterceptor_MissingAuth(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-ignore-path",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := inter(fakeServer, &FakeServerStream{
		ctx: metadata.NewIncomingContext(context.TODO(), metadata.New(map[string]string{})),
	}, info, handler)

	assert.NotNil(t, err)
	sts, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
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
