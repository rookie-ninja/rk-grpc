// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnaryClientInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryClientInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcbasic.ToOptionsKey(rkgrpcbasic.RkEntryNameValue, rkgrpcbasic.RpcTypeUnaryClient)])
}

func TestUnaryClientInterceptor_HappyCase(t *testing.T) {
	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeUnaryClient)])
}

func TestStreamClientInterceptor_WithoutOptions(t *testing.T) {
	inter := StreamClientInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcbasic.ToOptionsKey(rkgrpcbasic.RkEntryNameValue, rkgrpcbasic.RpcTypeStreamClient)])
}

func TestStreamClientInterceptor_HappyCase(t *testing.T) {
	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeStreamClient)])
}
