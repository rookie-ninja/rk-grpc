// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-prom"
	"google.golang.org/grpc/status"
	"strings"
)

var (
	DefaultLabelKeys = []string{
		"entryName",
		"entryType",
		"realm",
		"region",
		"az",
		"domain",
		"instance",
		"appVersion",
		"appName",
		"grpcService",
		"grpcMethod",
		"restMethod",
		"restPath",
		"grpcType",
		"resCode",
	}
)

const (
	ElapsedNano = "elapsedNano"
	Errors      = "errors"
	ResCode     = "resCode"
)

// Register bellow metrics into metrics set.
// 1: Request elapsed time with summary.
// 2: Error count with counter.
// 3: ResCode count with counter.
func initMetrics(set *optionSet) {
	// Ignoring duplicate metrics registration.
	// We don't want to break process because of it.
	set.MetricsSet.RegisterSummary(ElapsedNano, rkprom.SummaryObjectives, DefaultLabelKeys...)
	set.MetricsSet.RegisterCounter(Errors, DefaultLabelKeys...)
	set.MetricsSet.RegisterCounter(ResCode, DefaultLabelKeys...)
}

// Interceptor would distinguish loggers based on.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type nad options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:  rkgrpcinter.RpcEntryNameValue,
		EntryType:  rkgrpcinter.RpcEntryTypeValue,
		registerer: prometheus.DefaultRegisterer,
	}

	for i := range opts {
		opts[i](set)
	}

	namespace := strings.ReplaceAll(rkentry.GlobalAppCtx.GetAppInfoEntry().AppName, "-", "_")
	subSystem := strings.ReplaceAll(set.EntryName, "-", "_")
	set.MetricsSet = rkprom.NewMetricsSet(
		namespace,
		subSystem,
		set.registerer)

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	initMetrics(set)

	return set
}

// Options which is used while initializing logging interceptor
type optionSet struct {
	EntryName  string
	EntryType  string
	registerer prometheus.Registerer
	MetricsSet *rkprom.MetricsSet
}

type Option func(*optionSet)

// Provide entry name and type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}

// Provide prometheus registerer.
func WithRegisterer(registerer prometheus.Registerer) Option {
	return func(set *optionSet) {
		if registerer != nil {
			set.registerer = registerer
		}
	}
}

// Get option set from context
func getOptionSet(ctx context.Context) *optionSet {
	entryName := rkgrpcctx.GetEntryName(ctx)
	rpcType := rkgrpcctx.GetRpcType(ctx)
	return optionsMap[rkgrpcinter.ToOptionsKey(entryName, rpcType)]
}

// Get duration metrics.
func GetDurationMetrics(ctx context.Context) prometheus.Observer {
	if ctx == nil {
		return nil
	}

	if metricsSet := GetMetricsSet(ctx); metricsSet != nil {
		return metricsSet.GetSummaryWithValues(ElapsedNano, getValues(ctx)...)
	}

	return nil
}

// Get error metrics.
func GetErrorMetrics(ctx context.Context) prometheus.Counter {
	if ctx == nil {
		return nil
	}

	if metricsSet := GetMetricsSet(ctx); metricsSet != nil {
		return metricsSet.GetCounterWithValues(Errors, getValues(ctx)...)
	}

	return nil
}

// Get res code metrics.
func GetResCodeMetrics(ctx context.Context) prometheus.Counter {
	if ctx == nil {
		return nil
	}

	if metricsSet := GetMetricsSet(ctx); metricsSet != nil {
		return metricsSet.GetCounterWithValues(ResCode, getValues(ctx)...)
	}

	return nil
}

// Get metrics set.
func GetMetricsSet(ctx context.Context) *rkprom.MetricsSet {
	if val := getOptionSet(ctx); val != nil {
		return val.MetricsSet
	}

	return nil
}

// Metrics set already set into context.
func getValues(ctx context.Context) []string {
	method := rkgrpcctx.GetMethodName(ctx)
	rpcType := rkgrpcctx.GetRpcType(ctx)
	err := rkgrpcctx.GetError(ctx)

	entryName, entryType, resCode := "", "", ""
	if set := getOptionSet(ctx); set != nil {
		entryName = set.EntryName
		entryType = set.EntryType
		resCode = status.Code(err).String()
	}

	grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(method)
	gwMethod, gwPath, _, _ := rkgrpcinter.GetGwInfo(rkgrpcctx.GetIncomingHeaders(ctx))

	values := []string{
		entryName,
		entryType,
		rkgrpcinter.Realm.String,
		rkgrpcinter.Region.String,
		rkgrpcinter.AZ.String,
		rkgrpcinter.Domain.String,
		rkgrpcinter.LocalHostname.String,
		rkentry.GlobalAppCtx.GetAppInfoEntry().Version,
		rkentry.GlobalAppCtx.GetAppInfoEntry().AppName,
		grpcService,
		grpcMethod,
		gwMethod,
		gwPath,
		rpcType,
		resCode,
	}

	return values
}
