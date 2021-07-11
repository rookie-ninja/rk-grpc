// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpctrace

import (
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestGrpcMetadataCarrier_Get_HappyCase(t *testing.T) {
	key, value := "ut-key", "ut-value"
	md := metadata.Pairs(key, value)
	carrier := &rkgrpcctx.GrpcMetadataCarrier{
		Md: &md,
	}

	assert.Equal(t, value, carrier.Get(key))
}

func TestGrpcMetadataCarrier_Set_HappyCase(t *testing.T) {
	key, value := "ut-key", "ut-value"
	md := metadata.Pairs()
	carrier := &rkgrpcctx.GrpcMetadataCarrier{
		Md: &md,
	}

	carrier.Set(key, value)
	assert.Equal(t, value, carrier.Get(key))
}

func TestGrpcMetadataCarrier_Keys_HappyCase(t *testing.T) {
	key1, value1 := "ut-key-1", "ut-value-1"
	key2, value2 := "ut-key-2", "ut-value-2"

	md := metadata.Pairs()
	carrier := &rkgrpcctx.GrpcMetadataCarrier{
		Md: &md,
	}

	carrier.Set(key1, value1)
	carrier.Set(key2, value2)

	assert.Len(t, carrier.Keys(), 2)
}

func TestCreateFileExporter_WithEmptyOutputPath(t *testing.T) {
	exporter := CreateFileExporter("")
	assert.NotNil(t, exporter)
}

func TestCreateFileExporter_WithStdout(t *testing.T) {
	exporter := CreateFileExporter("stdout")
	assert.NotNil(t, exporter)
}

func TestCreateFileExporter_WithFilepath(t *testing.T) {
	exporter := CreateFileExporter("logs/tracing.log")
	assert.NotNil(t, exporter)
}

func TestCreateJaegerExporter_WithEmptyEndpoint(t *testing.T) {
	exporter := CreateJaegerExporter("", "", "")
	assert.NotNil(t, exporter)
}

func TestCreateJaegerExporter_HappyCase(t *testing.T) {
	exporter := CreateJaegerExporter("localhost:1949", "user", "pass")
	assert.NotNil(t, exporter)
}

func TestWithExporter_WithNilInput(t *testing.T) {
	opt := WithExporter(nil)
	exporter, _ := stdout.NewExporter()
	set := &optionSet{
		Exporter: exporter,
	}
	opt(set)

	assert.Equal(t, exporter, set.Exporter)
}

func TestWithExporter_HappyCase(t *testing.T) {
	exporter, _ := stdout.NewExporter()

	opt := WithExporter(exporter)
	set := &optionSet{}
	opt(set)

	assert.Equal(t, exporter, set.Exporter)
}

func TestWithSpanProcessor_WithNilInput(t *testing.T) {
	opt := WithSpanProcessor(nil)

	exporter, _ := stdout.NewExporter()
	processor := sdktrace.NewSimpleSpanProcessor(exporter)
	set := &optionSet{
		Processor: processor,
	}
	opt(set)

	assert.Equal(t, processor, set.Processor)
}

func TestWithSpanProcessor_HappyCase(t *testing.T) {
	exporter, _ := stdout.NewExporter()
	processor := sdktrace.NewSimpleSpanProcessor(exporter)

	opt := WithSpanProcessor(processor)

	set := &optionSet{}
	opt(set)

	assert.Equal(t, processor, set.Processor)
}

func TestWithTracerProvider_WithNilInput(t *testing.T) {
	opt := WithTracerProvider(nil)

	provider := sdktrace.NewTracerProvider()
	set := &optionSet{
		Provider: provider,
	}
	opt(set)

	assert.Equal(t, provider, set.Provider)
}

func TestWithTracerProvider_HappyCase(t *testing.T) {
	provider := sdktrace.NewTracerProvider()

	opt := WithTracerProvider(provider)

	set := &optionSet{}
	opt(set)

	assert.Equal(t, provider, set.Provider)
}

func TestWithPropagator_WithNilInput(t *testing.T) {
	opt := WithPropagator(nil)

	propagator := propagation.NewCompositeTextMapPropagator()
	set := &optionSet{
		Propagator: propagator,
	}
	opt(set)

	assert.Equal(t, propagator, set.Propagator)
}

func TestWithPropagator_HappyCase(t *testing.T) {
	propagator := propagation.NewCompositeTextMapPropagator()

	opt := WithPropagator(propagator)

	set := &optionSet{}
	opt(set)

	assert.Equal(t, propagator, set.Propagator)
}

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	entryName, entryType := "ut-entry-name", "ut-entry-type"

	opt := WithEntryNameAndType(entryName, entryType)

	set := &optionSet{}
	opt(set)

	assert.Equal(t, entryName, set.EntryName)
	assert.Equal(t, entryType, set.EntryType)
}
