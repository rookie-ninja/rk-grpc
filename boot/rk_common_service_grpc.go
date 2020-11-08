// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc

import (
	"context"
	"github.com/golang/protobuf/jsonpb"
	"github.com/rookie-ninja/rk-common/info"
	"github.com/rookie-ninja/rk-grpc/boot/api/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
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
		MemStatsAfterGc: &rk_grpc_common_v1.MemStats{},
	}

	jsonpb.UnmarshalString(before, res.MemStatsBeforeGc)
	jsonpb.UnmarshalString(after, res.MemStatsAfterGc)

	return res, nil
}

// DumpConfig Stub
func (service *CommonServiceGRpc) DumpConfig(ctx context.Context, request *rk_grpc_common_v1.DumpConfigRequest) (*rk_grpc_common_v1.DumpConfigResponse, error) {
	// Add auto generated request ID
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	res := &rk_grpc_common_v1.DumpConfigResponse{
		Viper: make([]*rk_grpc_common_v1.Viper, 0),
		Rk: make([]*rk_grpc_common_v1.RK, 0),
	}

	// viper
	vp := rk_info.ViperConfigToStruct()
	for i := range vp {
		e := vp[i]
		res.Viper = append(res.Viper, &rk_grpc_common_v1.Viper{
			Name: e.Name,
			Raw: e.Raw,
		})
	}

	// rk
	rk := rk_info.RkConfigToStruct()
	for i := range rk {
		e := vp[i]
		res.Rk = append(res.Rk, &rk_grpc_common_v1.RK{
			Name: e.Name,
			Raw: e.Raw,
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

	println(rk_info.BasicInfoToJSONPretty())

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