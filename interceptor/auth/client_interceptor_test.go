// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcauth

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestUnaryClientInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		assert.Nil(t, ctx.Value(rkgrpcinter.RpcAuthorizationHeaderKey))
		assert.Nil(t, ctx.Value(rkgrpcinter.RpcApiKeyHeaderKey))
		return nil
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, "ut-method", fakeRequest, fakeResponse, cc, invoker)
}

func TestStreamClientInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		assert.Nil(t, ctx.Value(rkgrpcinter.RpcAuthorizationHeaderKey))
		assert.Nil(t, ctx.Value(rkgrpcinter.RpcApiKeyHeaderKey))
		return nil, nil
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, &grpc.StreamDesc{}, cc, "ut-method", streamer)
}
