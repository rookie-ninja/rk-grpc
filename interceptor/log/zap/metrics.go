// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc_log

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-prom"
)

var (
	serverMetrics    = initMetrics("grpc_server")
	clientMetrics    = initMetrics("grpc_client")
	defaultLabelKeys = []string{"realm", "region", "az", "domain", "app_version", "app_name", "path", "res_code"}
)

const (
	ElapsedNano         = "elapsed_nano"
	Errors              = "errors"
	BytesTransferredIn  = "bytes_transferred_in"
	BytesTransferredOut = "bytes_transferred_out"
	ResCode             = "res_code"
)

func initMetrics(subSystem string) *rk_prom.MetricsSet {
	metricsSet := rk_prom.NewMetricsSet("rk", subSystem)
	metricsSet.RegisterSummary(ElapsedNano, rk_prom.SummaryObjectives, defaultLabelKeys...)
	metricsSet.RegisterCounter(Errors, defaultLabelKeys...)
	metricsSet.RegisterCounter(BytesTransferredIn, defaultLabelKeys...)
	metricsSet.RegisterCounter(BytesTransferredOut, defaultLabelKeys...)
	metricsSet.RegisterCounter(ResCode, defaultLabelKeys...)

	return metricsSet
}

// Server related
func getServerDurationMetrics(method, resCode string) prometheus.Observer {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetSummaryWithValues(ElapsedNano, values...)
}

func getServerErrorMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(Errors, values...)
}

func getServerResCodeMetrics(method, inputResCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, inputResCode}
	return serverMetrics.GetCounterWithValues(ResCode, values...)
}

func getServerBytesTransInMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(BytesTransferredIn, values...)
}

func getServerBytesTransOutMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(BytesTransferredOut, values...)
}

// Client related
func getClientDurationMetrics(method, resCode string) prometheus.Observer {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetSummaryWithValues(ElapsedNano, values...)
}

func getClientErrorMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(Errors, values...)
}

func getClientResCodeMetrics(method, inputResCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, inputResCode}
	return clientMetrics.GetCounterWithValues(ResCode, values...)
}

func getClientBytesTransInMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(BytesTransferredIn, values...)
}

func getClientBytesTransOutMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(BytesTransferredOut, values...)
}

func GetServerMetricsSet() *rk_prom.MetricsSet {
	return serverMetrics
}

func GetClientMetricsSet() *rk_prom.MetricsSet {
	return clientMetrics
}

