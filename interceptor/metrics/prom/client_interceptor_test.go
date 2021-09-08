// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcmetrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestUnaryClientInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryClientInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeUnaryClient)])
}

func TestUnaryClientInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	reg := prometheus.NewRegistry()
	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithRegisterer(reg))

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, "ut-method", fakeRequest, fakeResponse, cc, invoker)
}

func TestStreamClientInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	reg := prometheus.NewRegistry()
	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithRegisterer(reg))

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return nil, nil
	}
	cc := &grpc.ClientConn{}
	ctx := context.TODO()

	inter(ctx, &grpc.StreamDesc{}, cc, "ut-method", streamer)
}
