// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnaryClientInterceptor_WithoutOptions(t *testing.T) {
	defer clearOptionsMap()
	inter := UnaryClientInterceptor()

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeUnaryClient)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func TestUnaryClientInterceptor_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	inter := UnaryClientInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeUnaryClient)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func TestStreamClientInterceptor_WithoutOptions(t *testing.T) {
	defer clearOptionsMap()
	inter := StreamClientInterceptor()

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeStreamClient)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func TestStreamClientInterceptor_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	inter := StreamClientInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeStreamClient)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}
