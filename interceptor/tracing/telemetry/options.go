// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpctrace is aa middleware of grpc framework for recording trace info of RPC
package rkgrpctrace

import (
	"context"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-logger"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"os"
	"path"
)

// NoopExporter exporter which do nothing
type NoopExporter struct{}

// ExportSpans handles export of SpanSnapshots by dropping them.
func (nsb *NoopExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

// Shutdown stops the exporter by doing nothing.
func (nsb *NoopExporter) Shutdown(context.Context) error { return nil }

// CreateNoopExporter Create a noop exporter
func CreateNoopExporter() sdktrace.SpanExporter {
	return &NoopExporter{}
}

// CreateFileExporter Create a file exporter whose default output is stdout.
func CreateFileExporter(outputPath string, opts ...stdouttrace.Option) sdktrace.SpanExporter {
	if opts == nil {
		opts = make([]stdouttrace.Option, 0)
	}

	if outputPath == "" {
		outputPath = "stdout"
	}

	if outputPath == "stdout" {
		opts = append(opts, stdouttrace.WithPrettyPrint())
	} else {
		// init lumberjack logger
		writer := rklogger.NewLumberjackConfigDefault()
		if !path.IsAbs(outputPath) {
			wd, _ := os.Getwd()
			outputPath = path.Join(wd, outputPath)
		}

		writer.Filename = outputPath

		opts = append(opts, stdouttrace.WithWriter(writer))
	}

	exporter, _ := stdouttrace.New(opts...)

	return exporter
}

// CreateJaegerExporter Create jaeger exporter with bellow condition.
//
// 1: If no option provided, then export to jaeger agent at localhost:6831
// 2: Jaeger agent
//    If no jaeger agent host was provided, then use localhost
//    If no jaeger agent port was provided, then use 6831
// 3: Jaeger collector
//    If no jaeger collector endpoint was provided, then use http://localhost:14268/api/traces
func CreateJaegerExporter(opt jaeger.EndpointOption) sdktrace.SpanExporter {
	// Assign default jaeger agent endpoint which is localhost:6831
	if opt == nil {
		opt = jaeger.WithAgentEndpoint()
	}

	exporter, err := jaeger.New(opt)

	if err != nil {
		rkcommon.ShutdownWithError(err)
	}

	return exporter
}

// Interceptor would distinguish logs set based on.
var optionsMap = make(map[string]*optionSet)

// Create an optionSet with rpc type.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcinter.RpcEntryNameValue,
		EntryType: rkgrpcinter.RpcEntryTypeValue,
	}

	for i := range opts {
		opts[i](set)
	}

	if set.Exporter == nil {
		set.Exporter = CreateNoopExporter()
	}

	if set.Processor == nil {
		set.Processor = sdktrace.NewBatchSpanProcessor(set.Exporter)
	}

	if set.Provider == nil {
		set.Provider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithSpanProcessor(set.Processor),
			sdktrace.WithResource(
				sdkresource.NewWithAttributes(
					semconv.SchemaURL,
					attribute.String("service.name", rkentry.GlobalAppCtx.GetAppInfoEntry().AppName),
					attribute.String("service.version", rkentry.GlobalAppCtx.GetAppInfoEntry().Version),
					attribute.String("service.entryName", set.EntryName),
					attribute.String("service.entryType", set.EntryType),
				)),
		)
	}

	set.Tracer = set.Provider.Tracer(set.EntryName, oteltrace.WithInstrumentationVersion(contrib.SemVersion()))

	if set.Propagator == nil {
		set.Propagator = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{})
	}

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing logging interceptor
type optionSet struct {
	EntryName  string
	EntryType  string
	Exporter   sdktrace.SpanExporter
	Processor  sdktrace.SpanProcessor
	Provider   *sdktrace.TracerProvider
	Propagator propagation.TextMapPropagator
	Tracer     oteltrace.Tracer
}

// Option parameters for newOptionSet
type Option func(*optionSet)

// WithExporter Provide sdktrace.SpanExporter.
func WithExporter(exporter sdktrace.SpanExporter) Option {
	return func(opt *optionSet) {
		if exporter != nil {
			opt.Exporter = exporter
		}
	}
}

// WithSpanProcessor Provide sdktrace.SpanProcessor.
func WithSpanProcessor(processor sdktrace.SpanProcessor) Option {
	return func(opt *optionSet) {
		if processor != nil {
			opt.Processor = processor
		}
	}
}

// WithTracerProvider Provide *sdktrace.TracerProvider.
func WithTracerProvider(provider *sdktrace.TracerProvider) Option {
	return func(opt *optionSet) {
		if provider != nil {
			opt.Provider = provider
		}
	}
}

// WithPropagator Provide propagation.TextMapPropagator.
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(opt *optionSet) {
		if propagator != nil {
			opt.Propagator = propagator
		}
	}
}

// WithEntryNameAndType Provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(opt *optionSet) {
		opt.EntryName = entryName
		opt.EntryType = entryType
	}
}

// ShutdownExporters Shutdown all exporters.
func ShutdownExporters() {
	for _, v := range optionsMap {
		v.Exporter.Shutdown(context.Background())
	}
}
