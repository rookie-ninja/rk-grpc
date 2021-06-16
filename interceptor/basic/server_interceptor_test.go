// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey(RkEntryNameValue, RpcTypeUnaryServer)])
}

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey("ut-entry-name", RpcTypeUnaryServer)])
}

func TestStreamServerInterceptor_WithoutOptions(t *testing.T) {
	inter := StreamServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey(RkEntryNameValue, RpcTypeStreamServer)])
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey("ut-entry-name", RpcTypeStreamServer)])
}

func TestGetGrpcInfo_HappyCase(t *testing.T) {
	md := metadata.Pairs(
		"x-forwarded-method", "ut-gw-method",
		"x-forwarded-path", "ut-gw-path",
		"x-forwarded-scheme", "ut-gw-scheme",
		"x-forwarded-user-agent", "ut-gw-user-agent",
	)

	fullMethod := "ut-grpc-service/ut-grpc-method"
	grpcService, grpcMethod := getGrpcInfo(fullMethod)
	gwMethod, gwPath, gwScheme, gwUserAgent := getGwInfo(md)

	assert.Equal(t, "ut-gw-method", gwMethod)
	assert.Equal(t, "ut-gw-path", gwPath)
	assert.Equal(t, "ut-gw-scheme", gwScheme)
	assert.Equal(t, "ut-gw-user-agent", gwUserAgent)
	assert.Equal(t, "ut-grpc-service", grpcService)
	assert.Equal(t, "ut-grpc-method", grpcMethod)
}
