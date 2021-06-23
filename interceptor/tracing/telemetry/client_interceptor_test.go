// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpctrace

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"testing"
)

func TestUnaryClientInterceptor_HappyCase(t *testing.T) {
	exporter, _ := stdout.NewExporter()
	processor := sdktrace.NewSimpleSpanProcessor(exporter)
	provider := sdktrace.NewTracerProvider()
	propagator := propagation.NewCompositeTextMapPropagator()
	entryName, entryType := "ut-entry-name", "ut-entry"

	UnaryClientInterceptor(
		WithEntryNameAndType(entryName, entryType),
		WithExporter(exporter),
		WithSpanProcessor(processor),
		WithTracerProvider(provider),
		WithPropagator(propagator))

	set := optionsMap[rkgrpcinter.ToOptionsKey(entryName, rkgrpcinter.RpcTypeUnaryClient)]
	assert.NotNil(t, set)
	assert.Equal(t, exporter, set.Exporter)
	assert.Equal(t, processor, set.Processor)
	assert.Equal(t, provider, set.Provider)
	assert.Equal(t, propagator, set.Propagator)

	// clear optionsMap
	optionsMap = make(map[string]*optionSet)
}

func TestUnaryClientInterceptor_WithoutOptions(t *testing.T) {
	entryName, entryType := "ut-entry-name", "ut-entry"

	UnaryClientInterceptor(
		WithEntryNameAndType(entryName, entryType))

	set := optionsMap[rkgrpcinter.ToOptionsKey(entryName, rkgrpcinter.RpcTypeUnaryClient)]
	assert.NotNil(t, set)
	assert.NotNil(t, set.Exporter)
	assert.NotNil(t, set.Processor)
	assert.NotNil(t, set.Provider)

	// clear optionsMap
	optionsMap = make(map[string]*optionSet)
}

func TestStreamClientInterceptor_HappyCase(t *testing.T) {
	exporter, _ := stdout.NewExporter()
	processor := sdktrace.NewSimpleSpanProcessor(exporter)
	provider := sdktrace.NewTracerProvider()
	propagator := propagation.NewCompositeTextMapPropagator()
	entryName, entryType := "ut-entry-name", "ut-entry"

	StreamClientInterceptor(
		WithEntryNameAndType(entryName, entryType),
		WithExporter(exporter),
		WithSpanProcessor(processor),
		WithTracerProvider(provider),
		WithPropagator(propagator))

	set := optionsMap[rkgrpcinter.ToOptionsKey(entryName, rkgrpcinter.RpcTypeStreamClient)]
	assert.NotNil(t, set)
	assert.Equal(t, exporter, set.Exporter)
	assert.Equal(t, processor, set.Processor)
	assert.Equal(t, provider, set.Provider)
	assert.Equal(t, propagator, set.Propagator)

	// clear optionsMap
	optionsMap = make(map[string]*optionSet)
}

func TestStreamClientInterceptor_WithoutOptions(t *testing.T) {
	entryName, entryType := "ut-entry-name", "ut-entry"

	StreamClientInterceptor(
		WithEntryNameAndType(entryName, entryType))

	set := optionsMap[rkgrpcinter.ToOptionsKey(entryName, rkgrpcinter.RpcTypeStreamClient)]
	assert.NotNil(t, set)
	assert.NotNil(t, set.Exporter)
	assert.NotNil(t, set.Processor)
	assert.NotNil(t, set.Provider)
	assert.NotNil(t, set.Propagator)

	// clear optionsMap
	optionsMap = make(map[string]*optionSet)
}
