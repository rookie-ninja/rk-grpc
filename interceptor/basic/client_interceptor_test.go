// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnaryClientInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryClientInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey(RkEntryNameValue, RpcTypeUnaryClient)])
}

func TestUnaryClientInterceptor_HappyCase(t *testing.T) {
	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey("ut-entry-name", RpcTypeUnaryClient)])
}

func TestStreamClientInterceptor_WithoutOptions(t *testing.T) {
	inter := StreamClientInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey(RkEntryNameValue, RpcTypeStreamClient)])
}

func TestStreamClientInterceptor_HappyCase(t *testing.T) {
	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[ToOptionsKey("ut-entry-name", RpcTypeStreamClient)])
}
