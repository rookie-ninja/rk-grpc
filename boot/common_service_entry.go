// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	api "github.com/rookie-ninja/rk-grpc/boot/api/gen/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"net/http"
	"path"
	"runtime"
)

const (
	// CommonServiceEntryType default entry type
	CommonServiceEntryType = "GrpcCommonServiceEntry"
	// CommonServiceEntryNameDefault default entry name
	CommonServiceEntryNameDefault = "GrpcCommonServiceDefault"
	// CommonServiceEntryDescription default entry description
	CommonServiceEntryDescription = "Internal RK entry which implements commonly used API with grpc framework."
	// CommonServiceGwMappingFilePath default path for gw_mapping.yaml file
	CommonServiceGwMappingFilePath = "api/v1/gw_mapping.yaml"
)

// BootConfigCommonService Bootstrap config of common service.
// 1: Enabled: Enable common service.
type BootConfigCommonService struct {
	Enabled bool `yaml:"enabled"`
}

// CommonServiceEntry RK common service which contains commonly used APIs
// 1: Healthy GET Returns true if process is alive
// 2: Gc GET Trigger gc()
// 3: Info GET Returns entry basic information
// 4: Configs GET Returns viper configs in GlobalAppCtx
// 5: Apis GET Returns list of apis registered in gin router
// 6: Sys GET Returns CPU and Memory information
// 7: Req GET Returns request metrics
// 8: Certs GET Returns certificates
// 9: Entries GET Returns entries
type CommonServiceEntry struct {
	EntryName         string                    `json:"entryName" yaml:"entryName"`
	EntryType         string                    `json:"entryType" yaml:"entryType"`
	EntryDescription  string                    `json:"entryDescription" yaml:"entryDescription"`
	EventLoggerEntry  *rkentry.EventLoggerEntry `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	ZapLoggerEntry    *rkentry.ZapLoggerEntry   `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	RegFuncGrpc       GrpcRegFunc               `json:"regFuncGrpc" yaml:"regFuncGrpc"`
	RegFuncGw         GwRegFunc                 `json:"regFuncGw" yaml:"regFuncGw"`
	GwMappingFilePath string                    `json:"gwMappingFilePath" yaml:"gwMappingFilePath"`
	GwMapping         map[string]string         `json:"gwMapping" yaml:"gwMapping"`
}

// CommonServiceEntryOption Common service entry option function.
type CommonServiceEntryOption func(*CommonServiceEntry)

// WithNameCommonService Provide name.
func WithNameCommonService(name string) CommonServiceEntryOption {
	return func(entry *CommonServiceEntry) {
		entry.EntryName = name
	}
}

// WithEventLoggerEntryCommonService Provide rkentry.EventLoggerEntry.
func WithEventLoggerEntryCommonService(eventLoggerEntry *rkentry.EventLoggerEntry) CommonServiceEntryOption {
	return func(entry *CommonServiceEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// WithZapLoggerEntryCommonService Provide rkentry.ZapLoggerEntry.
func WithZapLoggerEntryCommonService(zapLoggerEntry *rkentry.ZapLoggerEntry) CommonServiceEntryOption {
	return func(entry *CommonServiceEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// NewCommonServiceEntry Create new common service entry with options.
func NewCommonServiceEntry(opts ...CommonServiceEntryOption) *CommonServiceEntry {
	entry := &CommonServiceEntry{
		EntryName:         CommonServiceEntryNameDefault,
		EntryType:         CommonServiceEntryType,
		EntryDescription:  CommonServiceEntryDescription,
		ZapLoggerEntry:    rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry:  rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		RegFuncGrpc:       registerRkCommonService,
		RegFuncGw:         api.RegisterRkCommonServiceHandlerFromEndpoint,
		GwMappingFilePath: CommonServiceGwMappingFilePath,
		GwMapping:         make(map[string]string),
	}

	for i := range opts {
		opts[i](entry)
	}

	return entry
}

// Bootstrap common service entry
func (entry *CommonServiceEntry) Bootstrap(ctx context.Context) {
	// No op
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"bootstrap",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	logger := entry.ZapLoggerEntry.GetLogger()

	if raw := ctx.Value(bootstrapEventIdKey); raw != nil {
		event.SetEventId(raw.(string))
		logger = logger.With(zap.String("eventId", event.GetEventId()))
	}

	entry.logBasicInfo(event)

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	logger.Info("Bootstrapping CommonServiceEntry.", event.ListPayloads()...)
}

// Interrupt common service entry
func (entry *CommonServiceEntry) Interrupt(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"interrupt",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	logger := entry.ZapLoggerEntry.GetLogger()

	if raw := ctx.Value(bootstrapEventIdKey); raw != nil {
		event.SetEventId(raw.(string))
		logger = logger.With(zap.String("eventId", event.GetEventId()))
	}

	entry.logBasicInfo(event)

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	logger.Info("Interrupting CommonServiceEntry.", event.ListPayloads()...)
}

// GetName Get name of entry.
func (entry *CommonServiceEntry) GetName() string {
	return entry.EntryName
}

// GetType Get entry type.
func (entry *CommonServiceEntry) GetType() string {
	return entry.EntryType
}

// String Stringfy entry.
func (entry *CommonServiceEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// MarshalJSON Marshal entry.
func (entry *CommonServiceEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":        entry.EntryName,
		"entryType":        entry.EntryType,
		"entryDescription": entry.EntryDescription,
		"zapLoggerEntry":   entry.ZapLoggerEntry.GetName(),
		"eventLoggerEntry": entry.EventLoggerEntry.GetName(),
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Not supported.
func (entry *CommonServiceEntry) UnmarshalJSON([]byte) error {
	return nil
}

// GetDescription Get description of entry.
func (entry *CommonServiceEntry) GetDescription() string {
	return entry.EntryDescription
}

func (entry *CommonServiceEntry) logBasicInfo(event rkquery.Event) {
	event.AddPayloads(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
	)
}

// Helper function of /healthy call.
func doHealthy(context.Context) *rkentry.HealthyResponse {
	return &rkentry.HealthyResponse{
		Healthy: true,
	}
}

// Healthy Stub.
func (entry *CommonServiceEntry) Healthy(ctx context.Context, request *api.HealthyRequest) (*structpb.Struct, error) {
	event := rkgrpcctx.GetEvent(ctx)

	event.AddPair("healthy", "true")

	return structpb.NewStruct(rkcommon.ConvertStructToMap(doHealthy(ctx)))
}

// Helper function of /gc.
func doGc(context.Context) *rkentry.GcResponse {
	before := rkentry.NewMemInfo()
	runtime.GC()
	after := rkentry.NewMemInfo()

	return &rkentry.GcResponse{
		MemStatBeforeGc: before,
		MemStatAfterGc:  after,
	}
}

// Gc Stub.
func (entry *CommonServiceEntry) Gc(ctx context.Context, request *api.GcRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doGc(ctx)))
}

// Helper function of /info.
func doInfo(context.Context) *rkentry.ProcessInfo {
	return rkentry.NewProcessInfo()
}

// Info Stub.
func (entry *CommonServiceEntry) Info(ctx context.Context, request *api.InfoRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doInfo(ctx)))
}

// Helper function of /configs.
func doConfigs(context.Context) *rkentry.ConfigsResponse {
	res := &rkentry.ConfigsResponse{
		Entries: make([]*rkentry.ConfigsResponse_ConfigEntry, 0),
	}

	for _, v := range rkentry.GlobalAppCtx.ListConfigEntries() {
		configEntry := &rkentry.ConfigsResponse_ConfigEntry{
			EntryName:        v.GetName(),
			EntryType:        v.GetType(),
			EntryDescription: v.GetDescription(),
			EntryMeta:        v.GetViperAsMap(),
			Path:             v.Path,
		}

		res.Entries = append(res.Entries, configEntry)
	}

	return res
}

// Configs Stub.
func (entry *CommonServiceEntry) Configs(ctx context.Context, request *api.ConfigsRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doConfigs(ctx)))
}

// Compose swagger URL based on SwEntry.
func getSwUrl(ctx context.Context, entry *GwEntry) string {
	if entry.IsSwEnabled() {
		scheme := "http"
		if entry.IsServerTlsEnabled() {
			scheme = "https"
		}

		remoteIp, _, _ := rkgrpcinter.GetRemoteAddressSet(ctx)

		return fmt.Sprintf("%s://%s:%d%s",
			scheme,
			remoteIp,
			entry.SwEntry.Port,
			entry.SwEntry.Path)
	}

	return ""
}

// Compose gateway related elements based on GwEntry and SwEntry.
func getGwMapping(ctx context.Context, entry *GrpcEntry, grpcMethod string) *rkentry.ApisResponse_Rest {
	res := &rkentry.ApisResponse_Rest{}

	if !entry.IsGwEnabled() {
		return res
	}

	var value *gwRule
	var ok bool
	if value, ok = entry.GwEntry.GwMapping[grpcMethod]; !ok {
		return res
	}

	res.Port = entry.GwEntry.HttpPort
	res.Method = value.Method
	res.Pattern = value.Pattern
	res.SwUrl = getSwUrl(ctx, entry.GwEntry)

	return res
}

func doApis(ctx context.Context) *rkentry.ApisResponse {
	res := &rkentry.ApisResponse{
		Entries: make([]*rkentry.ApisResponse_Entry, 0),
	}

	grpcEntry := getEntry(ctx)

	if grpcEntry == nil {
		return res
	}

	for serviceName, serviceInfo := range grpcEntry.Server.GetServiceInfo() {
		for i := range serviceInfo.Methods {
			method := serviceInfo.Methods[i]
			apiType := "Unary"
			if method.IsServerStream {
				apiType = "Stream"
			}

			entry := &rkentry.ApisResponse_Entry{
				EntryName: grpcEntry.GetName(),
				Grpc: &rkentry.ApisResponse_Grpc{
					Service: serviceName,
					Method:  method.Name,
					Port:    grpcEntry.Port,
					Type:    apiType,
					Gw:      getGwMapping(ctx, grpcEntry, serviceName+"."+method.Name),
				},
			}

			res.Entries = append(res.Entries, entry)
		}
	}

	return res
}

// Apis Stub
func (entry *CommonServiceEntry) Apis(ctx context.Context, request *api.ApisRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doApis(ctx)))
}

// Helper function of /sys
func doSys(context.Context) *rkentry.SysResponse {
	return &rkentry.SysResponse{
		CpuInfo:   rkentry.NewCpuInfo(),
		MemInfo:   rkentry.NewMemInfo(),
		NetInfo:   rkentry.NewNetInfo(),
		OsInfo:    rkentry.NewOsInfo(),
		GoEnvInfo: rkentry.NewGoEnvInfo(),
	}
}

// Sys Stub
func (entry *CommonServiceEntry) Sys(ctx context.Context, request *api.SysRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doSys(ctx)))
}

// Helper function for Req call
func doReq(ctx context.Context) *rkentry.ReqResponse {
	vector := rkgrpcmetrics.GetMetricsSet(ctx).GetSummary(rkgrpcmetrics.ElapsedNano)
	reqMetrics := rkentry.NewPromMetricsInfo(vector)

	// Fill missed metrics
	type innerGrpcInfo struct {
		grpcService string
		grpcMethod  string
	}

	apis := make([]*innerGrpcInfo, 0)

	grpcEntry := GetGrpcEntry(rkgrpcctx.GetEntryName(ctx))

	if grpcEntry != nil {
		infos := grpcEntry.Server.GetServiceInfo()
		for serviceName, serviceInfo := range infos {
			for j := range serviceInfo.Methods {
				apis = append(apis, &innerGrpcInfo{
					grpcService: serviceName,
					grpcMethod:  serviceInfo.Methods[j].Name,
				})
			}
		}
	}

	// Add empty metrics into result
	for i := range apis {
		api := apis[i]
		contains := false
		// check whether api was in request metrics from prometheus
		for j := range reqMetrics {
			if reqMetrics[j].GrpcMethod == api.grpcMethod && reqMetrics[j].GrpcService == api.grpcService {
				contains = true
			}
		}

		if !contains {
			reqMetrics = append(reqMetrics, &rkentry.ReqMetricsRK{
				GrpcService: apis[i].grpcService,
				GrpcMethod:  apis[i].grpcMethod,
				ResCode:     make([]*rkentry.ResCodeRK, 0),
			})
		}
	}

	return &rkentry.ReqResponse{
		Metrics: reqMetrics,
	}
}

// Req Stub
func (entry *CommonServiceEntry) Req(ctx context.Context, request *api.ReqRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doReq(ctx)))
}

// Helper function of /entries
func doEntriesHelper(m map[string]rkentry.Entry, res *rkentry.EntriesResponse) {
	// Iterate entries and construct EntryElement
	for i := range m {
		entry := m[i]
		element := &rkentry.EntriesResponse_Entry{
			EntryName:        entry.GetName(),
			EntryType:        entry.GetType(),
			EntryDescription: entry.GetDescription(),
			EntryMeta:        entry,
		}

		if entries, ok := res.Entries[entry.GetType()]; ok {
			entries = append(entries, element)
		} else {
			res.Entries[entry.GetType()] = []*rkentry.EntriesResponse_Entry{element}
		}
	}
}

// Helper function of /entries
func doEntries(ctx context.Context) *rkentry.EntriesResponse {
	res := &rkentry.EntriesResponse{
		Entries: make(map[string][]*rkentry.EntriesResponse_Entry),
	}

	if ctx == nil {
		return res
	}

	// Iterate all internal and external entries in GlobalAppCtx
	doEntriesHelper(rkentry.GlobalAppCtx.ListEntries(), res)
	doEntriesHelper(rkentry.GlobalAppCtx.ListEventLoggerEntriesRaw(), res)
	doEntriesHelper(rkentry.GlobalAppCtx.ListZapLoggerEntriesRaw(), res)
	doEntriesHelper(rkentry.GlobalAppCtx.ListConfigEntriesRaw(), res)
	doEntriesHelper(rkentry.GlobalAppCtx.ListCertEntriesRaw(), res)
	doEntriesHelper(rkentry.GlobalAppCtx.ListCredEntriesRaw(), res)

	// App info entry
	appInfoEntry := rkentry.GlobalAppCtx.GetAppInfoEntry()
	res.Entries[appInfoEntry.GetType()] = []*rkentry.EntriesResponse_Entry{
		{
			EntryName:        appInfoEntry.GetName(),
			EntryType:        appInfoEntry.GetType(),
			EntryDescription: appInfoEntry.GetDescription(),
			EntryMeta:        appInfoEntry,
		},
	}

	return res
}

// Entries Stub
func (entry *CommonServiceEntry) Entries(ctx context.Context, request *api.EntriesRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doEntries(ctx)))
}

// Helper function of /entries
func doCerts(context.Context) *rkentry.CertsResponse {
	res := &rkentry.CertsResponse{
		Entries: make([]*rkentry.CertsResponse_Entry, 0),
	}

	entries := rkentry.GlobalAppCtx.ListCertEntries()

	// Iterator cert entries and construct CertResponse
	for i := range entries {
		entry := entries[i]

		certEntry := &rkentry.CertsResponse_Entry{
			EntryName:        entry.GetName(),
			EntryType:        entry.GetType(),
			EntryDescription: entry.GetDescription(),
		}

		if entry.Retriever != nil {
			certEntry.Endpoint = entry.Retriever.GetEndpoint()
			certEntry.Locale = entry.Retriever.GetLocale()
			certEntry.Provider = entry.Retriever.GetProvider()
			certEntry.ServerCertPath = entry.ServerCertPath
			certEntry.ServerKeyPath = entry.ServerKeyPath
			certEntry.ClientCertPath = entry.ClientCertPath
			certEntry.ClientKeyPath = entry.ClientKeyPath
		}

		if entry.Store != nil {
			certEntry.ServerCert = entry.Store.SeverCertString()
			certEntry.ClientCert = entry.Store.ClientCertString()
		}

		res.Entries = append(res.Entries, certEntry)
	}

	return res
}

// Certs Stub
func (entry *CommonServiceEntry) Certs(ctx context.Context, request *api.CertsRequest) (*structpb.Struct, error) {
	res, err := structpb.NewStruct(rkcommon.ConvertStructToMap(doCerts(ctx)))
	return res, err
}

// Helper function of /logs
func doLogsHelper(m map[string]rkentry.Entry, res *rkentry.LogsResponse) {
	entries := make([]*rkentry.LogsResponse_Entry, 0)

	// Iterate logger related entries and construct LogEntryElement
	for i := range m {
		entry := m[i]
		element := &rkentry.LogsResponse_Entry{
			EntryName:        entry.GetName(),
			EntryType:        entry.GetType(),
			EntryDescription: entry.GetDescription(),
			EntryMeta:        entry,
		}

		if val, ok := entry.(*rkentry.ZapLoggerEntry); ok {
			if val.LoggerConfig != nil {
				element.OutputPaths = val.LoggerConfig.OutputPaths
				element.ErrorOutputPaths = val.LoggerConfig.ErrorOutputPaths
			}
		}

		if val, ok := entry.(*rkentry.EventLoggerEntry); ok {
			if val.LoggerConfig != nil {
				element.OutputPaths = val.LoggerConfig.OutputPaths
				element.ErrorOutputPaths = val.LoggerConfig.ErrorOutputPaths
			}
		}

		entries = append(entries, element)
	}

	var entryType string

	if len(entries) > 0 {
		entryType = entries[0].EntryType
	}

	res.Entries[entryType] = entries
}

// Helper function of /logs
func doLogs(context.Context) *rkentry.LogsResponse {
	res := &rkentry.LogsResponse{
		Entries: make(map[string][]*rkentry.LogsResponse_Entry),
	}

	doLogsHelper(rkentry.GlobalAppCtx.ListEventLoggerEntriesRaw(), res)
	doLogsHelper(rkentry.GlobalAppCtx.ListZapLoggerEntriesRaw(), res)

	return res
}

// Logs Stub
func (entry *CommonServiceEntry) Logs(ctx context.Context, request *api.LogsRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doLogs(ctx)))
}

// Helper function of /git
func doGit(ctx context.Context) *rkentry.GitResponse {
	res := &rkentry.GitResponse{}

	if ctx == nil {
		return res
	}

	rkMetaEntry := rkentry.GlobalAppCtx.GetRkMetaEntry()
	if rkMetaEntry == nil {
		return res
	}

	res.Package = path.Base(rkMetaEntry.RkMeta.Git.Url)
	res.Branch = rkMetaEntry.RkMeta.Git.Branch
	res.Tag = rkMetaEntry.RkMeta.Git.Tag
	res.Url = rkMetaEntry.RkMeta.Git.Url
	res.CommitId = rkMetaEntry.RkMeta.Git.Commit.Id
	res.CommitIdAbbr = rkMetaEntry.RkMeta.Git.Commit.IdAbbr
	res.CommitSub = rkMetaEntry.RkMeta.Git.Commit.Sub
	res.CommitterName = rkMetaEntry.RkMeta.Git.Commit.Committer.Name
	res.CommitterEmail = rkMetaEntry.RkMeta.Git.Commit.Committer.Email
	res.CommitDate = rkMetaEntry.RkMeta.Git.Commit.Date

	return res
}

// Git Stub
func (entry *CommonServiceEntry) Git(ctx context.Context, request *api.GitRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doGit(ctx)))
}

// Helper function /deps
func doDeps(context.Context) *rkentry.DepResponse {
	res := &rkentry.DepResponse{}

	appInfoEntry := rkentry.GlobalAppCtx.GetAppInfoEntry()
	if appInfoEntry == nil {
		return res
	}
	res.GoMod = appInfoEntry.GoMod

	return res
}

// Deps Stub
func (entry *CommonServiceEntry) Deps(ctx context.Context, request *api.DepsRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doDeps(ctx)))
}

// Helper function /license
func doLicense(context.Context) *rkentry.LicenseResponse {
	res := &rkentry.LicenseResponse{}

	appInfoEntry := rkentry.GlobalAppCtx.GetAppInfoEntry()
	if appInfoEntry == nil {
		return res
	}

	res.License = appInfoEntry.License

	return res
}

// License Stub
func (entry *CommonServiceEntry) License(ctx context.Context, request *api.LicenseRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doLicense(ctx)))
}

// Helper function /readme
func doReadme(context.Context) *rkentry.ReadmeResponse {
	res := &rkentry.ReadmeResponse{}

	appInfoEntry := rkentry.GlobalAppCtx.GetAppInfoEntry()
	if appInfoEntry == nil {
		return res
	}

	res.Readme = appInfoEntry.Readme

	return res
}

// Readme Get README file contents.
func (entry *CommonServiceEntry) Readme(ctx context.Context, request *api.ReadmeRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doReadme(ctx)))
}

// GwErrorMapping Get error mapping file contents.
func (entry *CommonServiceEntry) GwErrorMapping(ctx context.Context, request *api.GwErrorMappingRequest) (*structpb.Struct, error) {
	return structpb.NewStruct(rkcommon.ConvertStructToMap(doGwErrorMapping(ctx)))
}

// Helper function /gwErrorMapping
func doGwErrorMapping(context.Context) *rkentry.GwErrorMappingResponse {
	res := &rkentry.GwErrorMappingResponse{
		Mapping: make(map[int32]*rkentry.GwErrorMappingResponse_Mapping),
	}

	// list grpc errors
	for k, v := range code.Code_name {
		element := &rkentry.GwErrorMappingResponse_Mapping{
			GrpcCode: k,
			GrpcText: v,
		}

		restCode := gwruntime.HTTPStatusFromCode(codes.Code(k))
		restText := http.StatusText(restCode)

		element.RestCode = int32(restCode)
		element.RestText = restText

		res.Mapping[element.GrpcCode] = element
	}

	return res
}

// Register common service
func registerRkCommonService(server *grpc.Server) {
	api.RegisterRkCommonServiceServer(server, NewCommonServiceEntry())
}

// Extract grpc entry from grpc_zap middleware
func getEntry(ctx context.Context) *GrpcEntry {
	if ctx == nil {
		return nil
	}

	entryRaw := rkentry.GlobalAppCtx.GetEntry(rkgrpcctx.GetEntryName(ctx))
	if entryRaw == nil {
		return nil
	}

	entry, _ := entryRaw.(*GrpcEntry)
	return entry
}
