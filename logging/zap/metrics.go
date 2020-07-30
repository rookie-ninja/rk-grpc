// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_logging_zap

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-prom"
)

var (
	serverMetrics    = initMetrics("server")
	clientMetrics    = initMetrics("client")
	defaultLabelKeys = []string{"realm", "region", "az", "domain", "app_version", "app_name", "method", "res_code"}
)

const (
	elapsedMS           = "elapsed_ms"
	errors              = "errors"
	bytesTransferredIn  = "bytes_transferred_in"
	bytesTransferredOut = "bytes_transferred_out"
	resCode             = "res_code"
)

func initMetrics(subSystem string) *rk_prom.MetricsSet {
	metricsSet := rk_prom.NewMetricsSet("rk", subSystem)
	metricsSet.RegisterSummary(elapsedMS, rk_prom.SummaryObjectives, defaultLabelKeys...)
	metricsSet.RegisterCounter(errors, defaultLabelKeys...)
	metricsSet.RegisterCounter(bytesTransferredIn, defaultLabelKeys...)
	metricsSet.RegisterCounter(bytesTransferredOut, defaultLabelKeys...)
	metricsSet.RegisterCounter(resCode, defaultLabelKeys...)

	return metricsSet
}

// Server related
func getServerDurationMetrics(method, resCode string) prometheus.Observer {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetSummaryWithValues(elapsedMS, values...)
}

func getServerErrorMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(errors, values...)
}

func getServerResCodeMetrics(method, inputResCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(resCode, values...)
}

func getServerBytesTransInMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(bytesTransferredIn, values...)
}

func getServerBytesTransOutMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return serverMetrics.GetCounterWithValues(bytesTransferredOut, values...)
}

// Client related
func getClientDurationMetrics(method, resCode string) prometheus.Observer {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetSummaryWithValues(elapsedMS, values...)
}

func getClientErrorMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(errors, values...)
}

func getClientResCodeMetrics(method, inputResCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(resCode, values...)
}

func getClientBytesTransInMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(bytesTransferredIn, values...)
}

func getClientBytesTransOutMetrics(method, resCode string) prometheus.Counter {
	values := []string{realm.String, region.String, az.String, domain.String, appVersion.String, appName, method, resCode}
	return clientMetrics.GetCounterWithValues(bytesTransferredOut, values...)
}
