// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	set := newOptionSet(rkgrpcctx.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, "ut-entry-name", set.EntryName)
	assert.Equal(t, "ut-entry", set.EntryType)
	assert.Equal(t, set,
		optionsMap[rkgrpcctx.ToOptionsKey("ut-entry-name", rkgrpcctx.RpcTypeUnaryServer)])

	clearInterceptorMetrics(set.MetricsSet)
}

func TestWithErrorToCode_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	errFunc := func(err error) codes.Code {
		return status.Code(err)
	}

	set := newOptionSet(rkgrpcctx.RpcTypeUnaryServer,
		WithErrorToCode(errFunc))

	assert.Equal(t,
		reflect.ValueOf(errFunc).Pointer(),
		reflect.ValueOf(set.ErrorToCodeFunc).Pointer())

	clearInterceptorMetrics(set.MetricsSet)
}

func TestWithRegisterer_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	registerer := prometheus.DefaultRegisterer
	set := newOptionSet(rkgrpcctx.RpcTypeUnaryServer,
		WithRegisterer(registerer))

	assert.Equal(t, set.registerer, registerer)
	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetOptionSet_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	set := GetOptionSet(nil)
	assert.Nil(t, set)
}

func TestGetOptionSet_WithoutRkContext(t *testing.T) {
	defer clearOptionsMap()
	set := GetOptionSet(context.TODO())
	assert.Nil(t, set)
}

func TestGetOptionSet_HappyCase(t *testing.T) {
	defer clearOptionsMap()
	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName("ut-entry-name"),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := newOptionSet(rkgrpcctx.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, set, GetOptionSet(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetDurationMetrics_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetDurationMetrics(nil))
}

func TestGetDurationMetrics_WithDefaultRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]

	assert.NotNil(t, set)
	assert.NotNil(t, GetDurationMetrics(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetDurationMetrics_WithCustomRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType),
		WithRegisterer(prometheus.NewRegistry()))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetDurationMetrics(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetErrorMetrics_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetErrorMetrics(nil))
}

func TestGetErrorMetrics_WithDefaultRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetErrorMetrics(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetErrorMetrics_WithCustomRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType),
		WithRegisterer(prometheus.NewRegistry()))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetErrorMetrics(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetResCodeMetrics_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetResCodeMetrics(nil))
}

func TestGetResCodeMetrics_WithDefaultRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetErrorMetrics(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetResCodeMetrics_WithCustomRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType),
		WithRegisterer(prometheus.NewRegistry()))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetResCodeMetrics(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetMetricsSet_WithNilContext(t *testing.T) {
	defer clearOptionsMap()
	assert.Nil(t, GetMetricsSet(nil))
}

func TestGetMetricsSet_WithDefaultRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetMetricsSet(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}

func TestGetMetricsSet_WithCustomRegisterer(t *testing.T) {
	defer clearOptionsMap()
	entryName := "ut-entry-name"
	entryType := "ut-entry"

	// Create unary server interceptor in order to init metrics.
	UnaryServerInterceptor(
		WithEntryNameAndType(entryName, entryType),
		WithRegisterer(prometheus.NewRegistry()))

	ctx := rkgrpcctx.ContextWithPayload(context.TODO(),
		rkgrpcctx.WithEntryName(entryName),
		rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
			Type: rkgrpcctx.RpcTypeUnaryServer,
		}))

	set := optionsMap[rkgrpcctx.ToOptionsKey(entryName, rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
	assert.NotNil(t, GetMetricsSet(ctx))

	clearInterceptorMetrics(set.MetricsSet)
}
