// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	rk_ctx "github.com/rookie-ninja/rk-common/context"
	"github.com/rookie-ninja/rk-common/info"
	rk_metrics "github.com/rookie-ninja/rk-common/metrics"
	"github.com/rookie-ninja/rk-grpc/boot/api/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	rk_grpc_log "github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/shirou/gopsutil/v3/cpu"
	"google.golang.org/grpc"
	"math"
	"reflect"
	"runtime"
)

type CommonServiceGRpc struct{}

func NewCommonServiceGRpc() *CommonServiceGRpc {
	service := &CommonServiceGRpc{}
	return service
}

// GC Stub
func (service *CommonServiceGRpc) GC(ctx context.Context, request *rk_grpc_common_v1.GCRequest) (*rk_grpc_common_v1.GCResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	before := rk_info.MemStatsToJSON()
	runtime.GC()
	after := rk_info.MemStatsToJSON()

	res := &rk_grpc_common_v1.GCResponse{
		MemStatsBeforeGc: &rk_grpc_common_v1.MemStats{},
		MemStatsAfterGc:  &rk_grpc_common_v1.MemStats{},
	}

	jsonpb.UnmarshalString(before, res.MemStatsBeforeGc)
	jsonpb.UnmarshalString(after, res.MemStatsAfterGc)

	return res, nil
}

// DumpConfig Stub
func (service *CommonServiceGRpc) Config(ctx context.Context, request *rk_grpc_common_v1.DumpConfigRequest) (*rk_grpc_common_v1.DumpConfigResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	res := &rk_grpc_common_v1.DumpConfigResponse{
		Viper: make([]*rk_grpc_common_v1.Viper, 0),
		Rk:    make([]*rk_grpc_common_v1.RK, 0),
	}

	// viper
	vp := rk_info.ViperConfigToStruct()
	for i := range vp {
		e := vp[i]
		res.Viper = append(res.Viper, &rk_grpc_common_v1.Viper{
			Name: e.Name,
			Raw:  e.Raw,
		})
	}

	// rk
	rk := rk_info.RkConfigToStruct()
	for i := range rk {
		e := rk[i]
		res.Rk = append(res.Rk, &rk_grpc_common_v1.RK{
			Name: e.Name,
			Raw:  e.Raw,
		})
	}

	return res, nil
}

// Info Stub
func (service *CommonServiceGRpc) Info(ctx context.Context, request *rk_grpc_common_v1.InfoRequest) (*rk_grpc_common_v1.InfoResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	res := &rk_grpc_common_v1.InfoResponse{
		Info: &rk_grpc_common_v1.Info{},
	}

	jsonpb.UnmarshalString(rk_info.BasicInfoToJSON(), res.Info)

	return res, nil
}

// Healthy Stub
func (service *CommonServiceGRpc) Healthy(ctx context.Context, request *rk_grpc_common_v1.HealthyRequest) (*rk_grpc_common_v1.HealthyResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)
	event := rk_grpc_ctx.GetEvent(ctx)

	event.AddPair("healthy", "true")

	res := &rk_grpc_common_v1.HealthyResponse{
		Healthy: true,
	}

	return res, nil
}

// API Stub
func (service *CommonServiceGRpc) APIS(ctx context.Context, request *rk_grpc_common_v1.ListAPIRequest) (*rk_grpc_common_v1.ListAPIResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	event := rk_grpc_ctx.GetEvent(ctx)

	event.AddPair("apis", "true")

	entries := rk_ctx.GlobalAppCtx.ListEntries()

	apiList := make([]*rk_grpc_common_v1.API, 0)
	for i := range entries {
		raw := entries[i]
		if raw.GetType() == "grpc" {
			entry := raw.(*GRpcEntry)
			api := &rk_grpc_common_v1.API{
				Name: entry.GetName(),
				Grpc: convertToGRPC(entry),
				Gw: convertToGW(entry),
			}
			apiList = append(apiList, api)
		}
	}

	res := &rk_grpc_common_v1.ListAPIResponse{
		Api: apiList,
	}

	return res, nil
}

// Sys Stub
func (service *CommonServiceGRpc) Sys(ctx context.Context, request *rk_grpc_common_v1.SysRequest) (*rk_grpc_common_v1.SysResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	event := rk_grpc_ctx.GetEvent(ctx)

	event.AddPair("sys", "true")

	var cpuPercentage, memPercentage float64

	cpuStat, _ := cpu.Percent(0, false)
	memStat := rk_info.MemStatsToStruct()
	for i := range cpuStat {
		cpuPercentage = math.Round(cpuStat[i]*100) / 100
	}

	memPercentage = math.Round(memStat.MemPercentage*100) / 100

	res := &rk_grpc_common_v1.SysResponse{
		CpuPercentage: float32(cpuPercentage),
		MemPercentage: float32(memPercentage),
		MemUsageMb: uint32(memStat.MemAllocByte / (1024 * 1024)),
		UpTime: rk_info.BasicInfoToStruct().UpTimeStr,
	}

	return res, nil
}

// Req Stub
func (service *CommonServiceGRpc) Req(ctx context.Context, request *rk_grpc_common_v1.ReqRequest) (*rk_grpc_common_v1.ReqResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	event := rk_grpc_ctx.GetEvent(ctx)

	event.AddPair("req", "true")

	vector := rk_grpc_log.GetServerMetricsSet().GetSummaryVec(rk_grpc_log.ElapsedNano)
	metrics := rk_metrics.GetRequestMetrics(vector)

	// fill missed metrics
	apis := make([]string, 0)
	entries := rk_ctx.GlobalAppCtx.ListEntries()
	for i := range entries {
		raw := entries[i]
		if raw.GetType() == "grpc" {
			entry := raw.(*GRpcEntry)
			infos := entry.GetServer().GetServiceInfo()
			for _, v := range infos {
				for j := range v.Methods {
					apis = append(apis, v.Methods[j].Name)
				}
			}
		}
	}
	for i := range apis {
		if !containsMetrics(apis[i], metrics) {
			metrics = append(metrics, &rk_metrics.ReqMetricsRK {
				Path: apis[i],
				ResCode: make([]*rk_metrics.ResCodeRK, 0),
			})
		}
	}

	// convert to pb
	var metricsRK = make([]*rk_grpc_common_v1.ReqMetricsRK, 0)
	for i := range metrics {
		metric := metrics[i]
		metricRK := &rk_grpc_common_v1.ReqMetricsRK {
			Path: metric.Path,
			ElapsedNanoP50: float32(metric.ElapsedNanoP50),
			ElapsedNanoP90: float32(metric.ElapsedNanoP90),
			ElapsedNanoP99: float32(metric.ElapsedNanoP99),
			ElapsedNanoP999: float32(metric.ElapsedNanoP999),
			Count: uint32(metric.Count),
		}

		resCodeRKList := make([]*rk_grpc_common_v1.ResCodeRK, 0)
		for j := range metric.ResCode {
			resCode := metric.ResCode[j]
			resCodeRK := &rk_grpc_common_v1.ResCodeRK{
				ResCode: resCode.ResCode,
				Count: uint32(resCode.Count),
			}
			resCodeRKList = append(resCodeRKList, resCodeRK)
		}

		metricRK.ResCode = resCodeRKList
		metricsRK = append(metricsRK, metricRK)
	}

	res := &rk_grpc_common_v1.ReqResponse{
		Metrics: metricsRK,
	}

	return res, nil
}

func convertToGRPC(entry *GRpcEntry) []*rk_grpc_common_v1.GRPC {
	res := make([]*rk_grpc_common_v1.GRPC, 0)

	for k, v := range entry.GetServer().GetServiceInfo() {
		for i := range v.Methods {
			method := v.Methods[i]
			element := &rk_grpc_common_v1.GRPC{
				Service: k,
				Method: method.Name,
				Type: gRpcMethodTypeToString(&method),
				Port: uint32(entry.GetPort()),
			}

			res = append(res, element)
		}
	}

	return res
}

func convertToGW(entry *GRpcEntry) []*rk_grpc_common_v1.GW {
	res := make([]*rk_grpc_common_v1.GW, 0)

	if entry.GetGWEntry() == nil {
		return res
	}

	// get handlers as map
	handlerMap := reflect.ValueOf(entry.GetGWEntry().gwMux).Elem().FieldByName("handlers")

	iter := handlerMap.MapRange()
	for iter.Next() {
		// 1: method
		// method would be the restful method like GET, PUT and etc
		method := iter.Key().String()

		// Value is a list of data structure which needs to be parsed
		len := iter.Value().Len()
		for i := 0; i < len; i++ {
			// pool represent as [v1 rk apis] whose elements can be concat as path
			pool := iter.Value().Index(i).FieldByName("pat").FieldByName("pool")
			// 2: path
			path := "/"
			for j := 0; j < pool.Len(); j++ {
				path += pool.Index(j).String() + "/"
			}

			// 3: port
			port := uint32(entry.GetGWEntry().GetHttpPort())

			// 4: swagger
			sw := ""
			if entry.GetGWEntry().getSWEntry() != nil {
				sw = fmt.Sprintf("localhost:%d%s",
					entry.GetGWEntry().GetHttpPort(),
					entry.GetGWEntry().getSWEntry().GetPath())
			}

			// construct element
			element := &rk_grpc_common_v1.GW{
				Method: method,
				Path: path,
				Port: port,
				Sw: sw,
			}
			res = append(res, element)
		}
	}

	return res
}

func gRpcMethodTypeToString(method *grpc.MethodInfo) string {
	if method.IsServerStream {
		return "stream"
	}

	return "unary"
}

func containsMetrics(api string, metrics []*rk_metrics.ReqMetricsRK) bool {
	for i := range metrics {
		if metrics[i].Path == api {
			return true
		}
	}

	return false
}
