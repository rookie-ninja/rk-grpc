package rk_logging_zap

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-prom"
)

var (

	serverPromMetrics = initServerPromMetrics()
	clientPromMetrics = initClientPromMetrics()
	defaultLabelKeys  = []string{
		"realm",
		"region",
		"az",
		"domain",
		"appVersion",
		"application",
		"hostname",
		"method",
		"server_code",
	}
)

const (
	serverElapsedMS           = "server_elapsed_ms"
	serverErrors              = "server_errors"
	serverBytesTransferredIn  = "server_bytes_transferred_in"
	serverBytesTransferredOut = "server_bytes_transferred_out"
	serverCode                = "server_code"

	clientElapsedMS           = "client_elapsed_ms"
	clientErrors              = "client_errors"
	clientBytesTransferredIn  = "client_bytes_transferred_in"
	clientBytesTransferredOut = "client_bytes_transferred_out"
	clientResCode             = "client_res_code"
)

// Server related
func initServerPromMetrics() *rk_prom.MetricsSet {
	metricsSet := rk_prom.NewMetricsSet("rk", "server")
	metricsSet.RegisterSummary(serverElapsedMS, rk_prom.SummaryObjectives, defaultLabelKeys...)
	metricsSet.RegisterCounter(serverErrors, defaultLabelKeys...)
	metricsSet.RegisterCounter(serverBytesTransferredIn, defaultLabelKeys...)
	metricsSet.RegisterCounter(serverBytesTransferredOut, defaultLabelKeys...)
	metricsSet.RegisterCounter(serverCode, defaultLabelKeys...)

	return metricsSet
}

func getServerDurationMetrics(method, resCode string) prometheus.Observer {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return serverPromMetrics.GetSummaryWithValues(serverElapsedMS, values...)
}

func getServerErrorMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return serverPromMetrics.GetCounterWithValues(serverErrors, values...)
}

func getServerResCodeMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return serverPromMetrics.GetCounterWithValues(serverCode, values...)
}

func getServerBytesTransInMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return serverPromMetrics.GetCounterWithValues(serverBytesTransferredIn, values...)
}

func getServerBytesTransOutMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return serverPromMetrics.GetCounterWithValues(serverBytesTransferredOut, values...)
}

// Client related
func initClientPromMetrics() *rk_prom.MetricsSet {
	metricsSet := rk_prom.NewMetricsSet("rk", "server")

	metricsSet.RegisterSummary(clientElapsedMS, rk_prom.SummaryObjectives, defaultLabelKeys...)
	metricsSet.RegisterCounter(clientErrors, defaultLabelKeys...)
	metricsSet.RegisterCounter(clientBytesTransferredIn, defaultLabelKeys...)
	metricsSet.RegisterCounter(clientBytesTransferredOut, defaultLabelKeys...)
	metricsSet.RegisterCounter(clientResCode, defaultLabelKeys...)

	return metricsSet
}

func getClientDurationMetrics(method, resCode string) prometheus.Observer {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return clientPromMetrics.GetSummaryWithValues(clientElapsedMS, values...)
}

func getClientErrorMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return clientPromMetrics.GetCounterWithValues(clientErrors, values...)
}

func getClientResCodeMetrics(method, code string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		code,
	}

	return clientPromMetrics.GetCounterWithValues(clientResCode, values...)
}

func getClientBytesTransInMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return clientPromMetrics.GetCounterWithValues(clientBytesTransferredIn, values...)
}

func getClientBytesTransOutMetrics(method, resCode string) prometheus.Counter {
	values := []string{
		realmField.String,
		regionField.String,
		azField.String,
		domainField.String,
		appVersionField.String,
		appNameField.String,
		localHostname.String,
		method,
		resCode,
	}

	return clientPromMetrics.GetCounterWithValues(clientBytesTransferredOut, values...)
}
