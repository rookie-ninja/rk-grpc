// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcctx.ToOptionsKey(rkgrpcctx.RkEntryNameValue, rkgrpcctx.RpcTypeUnaryServer)])
}

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcctx.ToOptionsKey("ut-entry-name", rkgrpcctx.RpcTypeUnaryServer)])
}

func TestStreamServerInterceptor_WithoutOptions(t *testing.T) {
	inter := StreamServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcctx.ToOptionsKey(rkgrpcctx.RkEntryNameValue, rkgrpcctx.RpcTypeStreamServer)])
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcctx.ToOptionsKey("ut-entry-name", rkgrpcctx.RpcTypeStreamServer)])
}

func TestParseRpcPath_HappyCase(t *testing.T) {
	ctx := metadata.NewIncomingContext(
		context.TODO(),
		metadata.Pairs(
			"x-forwarded-method", "ut-gw-method",
			"x-forwarded-path", "ut-gw-path"))

	fullMethod := "ut-grpc-service/ut-grpc-method"
	grpcService, grpcMethod, gwMethod, gwPath := parseRpcPath(ctx, fullMethod)

	assert.Equal(t, "ut-gw-method", gwMethod)
	assert.Equal(t, "ut-gw-path", gwPath)
	assert.Equal(t, "ut-grpc-service", grpcService)
	assert.Equal(t, "ut-grpc-method", grpcMethod)
}
