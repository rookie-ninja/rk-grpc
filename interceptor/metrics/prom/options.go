// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcmetrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-prom"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	unknown     = "unknown"
)

func initMetrics(set *optionSet) {
	// Ignoring duplicate metrics registration.
	// We don't want to break process because of it.
	set.MetricsSet.RegisterSummary(ElapsedNano, rkprom.SummaryObjectives, DefaultLabelKeys...)
	set.MetricsSet.RegisterCounter(Errors, DefaultLabelKeys...)
	set.MetricsSet.RegisterCounter(ResCode, DefaultLabelKeys...)
}

// Interceptor would distinguish loggers based on.
var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:       rkgrpcbasic.RkEntryNameValue,
		EntryType:       rkgrpcbasic.RkEntryTypeValue,
		registerer:      prometheus.DefaultRegisterer,
		ErrorToCodeFunc: errorToCodesFuncDefault,
	}

	for i := range opts {
		opts[i](set)
	}

	set.MetricsSet = rkprom.NewMetricsSet(
		rkentry.GlobalAppCtx.GetAppInfoEntry().AppName,
		set.EntryName,
		set.registerer)

	key := rkgrpcbasic.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	initMetrics(set)

	return set
}

// Options which is used while initializing logging interceptor
type optionSet struct {
	EntryName       string
	EntryType       string
	registerer      prometheus.Registerer
	MetricsSet      *rkprom.MetricsSet
	ErrorToCodeFunc func(err error) codes.Code
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

// Provide error to code function.
func WithErrorToCode(errorToCodeFunc func(err error) codes.Code) Option {
	return func(set *optionSet) {
		if errorToCodeFunc != nil {
			set.ErrorToCodeFunc = errorToCodeFunc
		}
	}
}

// Get option set from context
func GetOptionSet(ctx context.Context) *optionSet {
	entryName := rkgrpcctx.GetEntryName(ctx)

	info := rkgrpcctx.GetRpcInfo(ctx)

	if info != nil {
		return optionsMap[rkgrpcbasic.ToOptionsKey(entryName, info.Type)]
	}
	return nil
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
	if val := GetOptionSet(ctx); val != nil {
		return val.MetricsSet
	}

	return nil
}

// Metrics set already set into context.
func getValues(ctx context.Context) []string {
	var options = GetOptionSet(ctx)

	entryName, entryType := unknown, unknown
	if options != nil {
		entryName = options.EntryName
		entryType = options.EntryType
	}

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)
	resCode := options.ErrorToCodeFunc(rpcInfo.Err).String()

	values := []string{
		entryName,
		entryType,
		rkgrpcbasic.Realm.String,
		rkgrpcbasic.Region.String,
		rkgrpcbasic.AZ.String,
		rkgrpcbasic.Domain.String,
		rkgrpcbasic.LocalHostname.String,
		rkentry.GlobalAppCtx.GetAppInfoEntry().Version,
		rkentry.GlobalAppCtx.GetAppInfoEntry().AppName,
		rpcInfo.GrpcService,
		rpcInfo.GrpcMethod,
		rpcInfo.GwMethod,
		rpcInfo.GwPath,
		rpcInfo.Type,
		resCode,
	}

	return values
}

func errorToCodesFuncDefault(err error) codes.Code {
	return status.Code(err)
}
