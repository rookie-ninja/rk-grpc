// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnaryServerInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeUnaryServer)])
}

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeUnaryServer)])
}

func TestStreamServerInterceptor_WithoutOptions(t *testing.T) {
	inter := StreamServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeStreamServer)])
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeStreamServer)])
}
