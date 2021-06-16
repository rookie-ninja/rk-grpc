// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpctrace

import (
	"context"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-logger"
	"go.opentelemetry.io/contrib"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"log"
	"os"
	"path"
)

// Grpc metadata carrier which will carries tracing info into grpc metadata to server side.
type GrpcMetadataCarrier struct {
	md *metadata.MD
}

// Get value with key from grpc metadata.
func (carrier *GrpcMetadataCarrier) Get(key string) string {
	values := carrier.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set value with key into grpc metadata.
func (carrier *GrpcMetadataCarrier) Set(key string, value string) {
	carrier.md.Set(key, value)
}

// List keys in grpc metadata.
func (carrier *GrpcMetadataCarrier) Keys() []string {
	out := make([]string, 0, len(*carrier.md))
	for key := range *carrier.md {
		out = append(out, key)
	}
	return out
}

// Create a file exporter whose default output is stdout.
func CreateFileExporter(outputPath string, opts ...stdout.Option) sdktrace.SpanExporter {
	if opts == nil {
		opts = make([]stdout.Option, 0)
	}

	if outputPath == "" {
		outputPath = "stdout"
	}

	if outputPath == "stdout" {
		opts = append(opts,
			stdout.WithPrettyPrint(),
			stdout.WithoutMetricExport())
	} else {
		// init lumberjack logger
		writer := rklogger.NewLumberjackConfigDefault()
		if !path.IsAbs(outputPath) {
			wd, _ := os.Getwd()
			outputPath = path.Join(wd, outputPath)
		}

		writer.Filename = outputPath

		opts = append(opts, stdout.WithWriter(writer))
	}

	exporter, _ := stdout.NewExporter(opts...)

	return exporter
}

// TODO: Wait for opentelemetry update version of jeager exporter. Current exporter is not compatible with jaeger agent.
func CreateJaegerExporter(host, port string) sdktrace.SpanExporter {
	if len(host) < 1 {
		host = "localhost"
	}

	if len(port) < 1 {
		port = "6832"
	}

	exporter, err := jaeger.NewRawExporter(
		jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(host),
			jaeger.WithAgentPort(port),
			jaeger.WithLogger(log.New(os.Stdout, "", 0))))

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
		EntryName: rkgrpcbasic.RkEntryNameValue,
		EntryType: rkgrpcbasic.RkEntryTypeValue,
	}

	for i := range opts {
		opts[i](set)
	}

	if set.Exporter == nil {
		set.Exporter, _ = stdout.NewExporter(stdout.WithPrettyPrint())
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

	key := rkgrpcbasic.ToOptionsKey(set.EntryName, rpcType)
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

type Option func(*optionSet)

// Provide sdktrace.SpanExporter.
func WithExporter(exporter sdktrace.SpanExporter) Option {
	return func(opt *optionSet) {
		if exporter != nil {
			opt.Exporter = exporter
		}
	}
}

// Provide sdktrace.SpanProcessor.
func WithSpanProcessor(processor sdktrace.SpanProcessor) Option {
	return func(opt *optionSet) {
		if processor != nil {
			opt.Processor = processor
		}
	}
}

// Provide *sdktrace.TracerProvider.
func WithTracerProvider(provider *sdktrace.TracerProvider) Option {
	return func(opt *optionSet) {
		if provider != nil {
			opt.Provider = provider
		}
	}
}

// Provide propagation.TextMapPropagator.
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(opt *optionSet) {
		if propagator != nil {
			opt.Propagator = propagator
		}
	}
}

// Provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(opt *optionSet) {
		opt.EntryName = entryName
		opt.EntryType = entryType
	}
}

// Shutdown all exporters.
func ShutdownExporters() {
	for _, v := range optionsMap {
		v.Exporter.Shutdown(context.Background())
	}
}
