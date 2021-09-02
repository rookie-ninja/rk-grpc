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
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, "ut-entry-name", set.EntryName)
	assert.Equal(t, "ut-entry", set.EntryType)
	assert.Equal(t, set,
		optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeUnaryServer)])

	clearInterceptorMetrics(set.MetricsSet)
}

func TestWithRegisterer_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	registerer := prometheus.DefaultRegisterer
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithRegisterer(registerer))

	assert.Equal(t, set.registerer, registerer)
	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetOptionSet_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	set := getOptionSet(nil)
	assert.Nil(t, set)
}

func TestGetOptionSet_WithoutRkContext(t *testing.T) {
	defer clearOptionsMap()
	set := getOptionSet(context.TODO())
	assert.Nil(t, set)
}

func TestGetDurationMetrics_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetDurationMetrics(nil))
}

func TestGetErrorMetrics_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetErrorMetrics(nil))
}

func TestGetResCodeMetrics_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetResCodeMetrics(nil))
}

func TestGetMetricsSet_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetMetricsSet(nil))
}
