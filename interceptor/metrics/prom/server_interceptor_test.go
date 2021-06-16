// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-prom"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnaryServerInterceptor_WithoutOptions(t *testing.T) {
	defer clearOptionsMap()
	inter := UnaryServerInterceptor()

	assert.NotNil(t, inter)

	set := optionsMap[rkgrpcbasic.ToOptionsKey(rkgrpcbasic.RkEntryNameValue, rkgrpcbasic.RpcTypeUnaryServer)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"),
		WithRegisterer(prometheus.NewRegistry()),
		WithErrorToCode(errorToCodesFuncDefault))

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeUnaryServer)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func TestStreamServerInterceptor_WithoutOptions(t *testing.T) {
	defer clearOptionsMap()
	inter := StreamServerInterceptor()

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcbasic.ToOptionsKey(rkgrpcbasic.RkEntryNameValue, rkgrpcbasic.RpcTypeStreamServer)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"),
		WithRegisterer(prometheus.NewRegistry()),
		WithErrorToCode(errorToCodesFuncDefault))

	assert.NotNil(t, inter)
	set := optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeStreamServer)]
	assert.NotNil(t, set)

	clearInterceptorMetrics(set.MetricsSet)
}

func clearInterceptorMetrics(set *rkprom.MetricsSet) {
	if set == nil {
		return
	}

	// Clear counters
	set.UnRegisterCounter(Errors)
	set.UnRegisterCounter(ResCode)

	// Clear summary
	set.UnRegisterSummary(ElapsedNano)
}

func clearOptionsMap() {
	for k := range optionsMap {
		delete(optionsMap, k)
	}
}
