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

func TestWithEntryNameAndType(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)

	defer clearAllMetrics()
}

func TestWithRegisterer(t *testing.T) {
	reg := prometheus.NewRegistry()
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithRegisterer(reg))

	assert.Equal(t, reg, set.registerer)

	defer clearAllMetrics()
}

func TestGetOptionSet(t *testing.T) {
	// With nil context
	assert.Nil(t, getOptionSet(nil))

	// Happy case
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"))

	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, "ut-entry")
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)

	assert.Equal(t, set, getOptionSet(ctx))

	defer clearAllMetrics()
}

func TestGetMetricsSet(t *testing.T) {
	reg := prometheus.NewRegistry()
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithRegisterer(reg))

	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, "ut-entry")
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)

	assert.Equal(t, set.MetricsSet, GetMetricsSet(ctx))

	defer clearAllMetrics()
}

func TestGetServerResCodeMetrics(t *testing.T) {
	// With nil context
	assert.Nil(t, GetResCodeMetrics(nil))

	// Happy case
	reg := prometheus.NewRegistry()
	newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithRegisterer(reg))

	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, "ut-entry")
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)

	assert.NotNil(t, GetResCodeMetrics(ctx))

	defer clearAllMetrics()
}

func TestGetErrorMetrics(t *testing.T) {
	// With nil context
	assert.Nil(t, GetErrorMetrics(nil))

	// Happy case
	reg := prometheus.NewRegistry()
	newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithRegisterer(reg))

	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, "ut-entry")
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)

	assert.NotNil(t, GetErrorMetrics(ctx))

	defer clearAllMetrics()
}

func TestGetDurationMetrics(t *testing.T) {
	// With nil context
	assert.Nil(t, GetDurationMetrics(nil))

	// Happy case
	reg := prometheus.NewRegistry()
	newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithRegisterer(reg))

	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, "ut-entry")
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)

	assert.NotNil(t, GetDurationMetrics(ctx))

	defer clearAllMetrics()
}
