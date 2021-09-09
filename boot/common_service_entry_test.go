// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	api "github.com/rookie-ninja/rk-grpc/boot/api/gen/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"reflect"
	"strings"
	"testing"
)

func TestWithNameCommonService_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	name := "ut-common-service"
	opt := WithNameCommonService(name)
	opt(entry)

	assert.Equal(t, name, entry.GetName())
}

func TestWithNameCommonService_WithEmptyName(t *testing.T) {
	entry := NewCommonServiceEntry()

	opt := WithNameCommonService("")
	opt(entry)

	assert.Equal(t, "", entry.GetName())
}

func TestWithEventLoggerEntryCommonService_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()

	opt := WithEventLoggerEntryCommonService(eventLoggerEntry)
	opt(entry)

	assert.Equal(t, eventLoggerEntry, entry.EventLoggerEntry)
}

func TestWithEventLoggerEntryCommonService_WithNilLogger(t *testing.T) {
	entry := NewCommonServiceEntry()

	opt := WithEventLoggerEntryCommonService(nil)
	opt(entry)

	assert.Nil(t, entry.EventLoggerEntry)
}

func TestWithZapLoggerEntryCommonService_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()

	opt := WithZapLoggerEntryCommonService(zapLoggerEntry)
	opt(entry)

	assert.Equal(t, zapLoggerEntry, entry.ZapLoggerEntry)
}

func TestWithZapLoggerEntryCommonService_WithNilLogger(t *testing.T) {
	entry := NewCommonServiceEntry()

	opt := WithZapLoggerEntryCommonService(nil)
	opt(entry)

	assert.Nil(t, entry.ZapLoggerEntry)
}

func TestNewCommonServiceEntry_WithoutOptions(t *testing.T) {
	entry := NewCommonServiceEntry()

	assert.NotNil(t, entry)
	assert.Equal(t, CommonServiceEntryNameDefault, entry.EntryName)
	assert.Equal(t, CommonServiceEntryType, entry.EntryType)
	assert.Equal(t, CommonServiceEntryDescription, entry.EntryDescription)
	assert.NotNil(t, entry.EventLoggerEntry)
	assert.NotNil(t, entry.ZapLoggerEntry)
	assert.Equal(t, reflect.ValueOf(registerRkCommonService).Pointer(), reflect.ValueOf(entry.RegFuncGrpc).Pointer())
	assert.Equal(t, reflect.ValueOf(api.RegisterRkCommonServiceHandlerFromEndpoint).Pointer(),
		reflect.ValueOf(entry.RegFuncGw).Pointer())
	assert.Equal(t, CommonServiceGwMappingFilePath, entry.GwMappingFilePath)
	assert.NotNil(t, entry.GwMapping)
}

func TestNewCommonServiceEntry_WithOptions(t *testing.T) {
	name := "ut-common-service"
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()

	entry := NewCommonServiceEntry(
		WithNameCommonService(name),
		WithEventLoggerEntryCommonService(eventLoggerEntry),
		WithZapLoggerEntryCommonService(zapLoggerEntry))

	assert.NotNil(t, entry)
	assert.Equal(t, name, entry.EntryName)
	assert.Equal(t, CommonServiceEntryType, entry.EntryType)
	assert.Equal(t, CommonServiceEntryDescription, entry.EntryDescription)
	assert.Equal(t, eventLoggerEntry, entry.EventLoggerEntry)
	assert.Equal(t, zapLoggerEntry, entry.ZapLoggerEntry)
	assert.Equal(t, reflect.ValueOf(registerRkCommonService).Pointer(), reflect.ValueOf(entry.RegFuncGrpc).Pointer())
	assert.Equal(t, reflect.ValueOf(api.RegisterRkCommonServiceHandlerFromEndpoint).Pointer(),
		reflect.ValueOf(entry.RegFuncGw).Pointer())
	assert.Equal(t, CommonServiceGwMappingFilePath, entry.GwMappingFilePath)
	assert.NotNil(t, entry.GwMapping)
}

func TestCommonServiceEntry_Bootstrap_HappyCase(t *testing.T) {
	assertNotPanic(t)

	entry := NewCommonServiceEntry(
		WithEventLoggerEntryCommonService(rkentry.NoopEventLoggerEntry()),
		WithZapLoggerEntryCommonService(rkentry.NoopZapLoggerEntry()))

	ctx := context.WithValue(context.Background(), bootstrapEventIdKey, "ut")
	entry.Bootstrap(ctx)
}

func TestCommonServiceEntry_Interrupt_HappyCase(t *testing.T) {
	assertNotPanic(t)

	entry := NewCommonServiceEntry(
		WithEventLoggerEntryCommonService(rkentry.NoopEventLoggerEntry()),
		WithZapLoggerEntryCommonService(rkentry.NoopZapLoggerEntry()))

	ctx := context.WithValue(context.Background(), bootstrapEventIdKey, "ut")
	entry.Interrupt(ctx)
}

func TestCommonServiceEntry_GetName_HappyCase(t *testing.T) {
	name := "ut-common-service"
	entry := NewCommonServiceEntry(WithNameCommonService(name))
	assert.Equal(t, name, entry.GetName())
}

func TestCommonServiceEntry_GetType_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()
	assert.Equal(t, CommonServiceEntryType, entry.GetType())
}

func TestCommonServiceEntry_String(t *testing.T) {
	entry := NewCommonServiceEntry()
	assert.Contains(t, entry.String(), "entryName")
	assert.Contains(t, entry.String(), "entryType")
	assert.Contains(t, entry.String(), "entryDescription")
	assert.Contains(t, entry.String(), "zapLoggerEntry")
	assert.Contains(t, entry.String(), "eventLoggerEntry")
}

func TestCommonServiceEntry_MarshalJSON_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	bytes, err := entry.MarshalJSON()
	assert.Nil(t, err)
	assert.NotEmpty(t, bytes)

	str := string(bytes)
	assert.Contains(t, str, "entryName")
	assert.Contains(t, str, "entryType")
	assert.Contains(t, str, "entryDescription")
	assert.Contains(t, str, "zapLoggerEntry")
	assert.Contains(t, str, "eventLoggerEntry")
}

func TestCommonServiceEntry_UnmarshalJSON_HappyCase(t *testing.T) {
	assertNotPanic(t)
	entry := NewCommonServiceEntry()
	entry.UnmarshalJSON(nil)
}

func TestCommonServiceEntry_GetDescription_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()
	assert.Equal(t, CommonServiceEntryDescription, entry.GetDescription())
}

func TestCommonServiceEntry_logBasicInfo(t *testing.T) {
	entry := NewCommonServiceEntry()

	event := rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEvent()
	entry.logBasicInfo(event)

	assert.Len(t, event.ListPayloads(), 2)
}

func TestCommonServiceEntry_doHealthy_HappyCase(t *testing.T) {
	resp := doHealthy(nil)

	assert.NotNil(t, resp)
	assert.True(t, resp.Healthy)
}

func TestCommonServiceEntry_Healthy_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	resp, err := entry.Healthy(context.TODO(), &api.HealthyRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.AsMap()["healthy"].(bool))
}

func TestCommonServiceEntry_doGc_HappyCase(t *testing.T) {
	resp := doGc(nil)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.MemStatBeforeGc)
	assert.NotNil(t, resp.MemStatAfterGc)
}

func TestCommonServiceEntry_Gc_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	resp, err := entry.Gc(context.TODO(), &api.GcRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.AsMap(), "memStatBeforeGc")
	assert.Contains(t, resp.AsMap(), "memStatAfterGc")
}

func TestCommonServiceEntry_doInfo_HappyCase(t *testing.T) {
	resp := doInfo(nil)

	assert.NotNil(t, resp)
	assert.IsType(t, &rkentry.ProcessInfo{}, resp)
}

func TestCommonServiceEntry_Info_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	resp, err := entry.Info(context.TODO(), &api.InfoRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_doConfigs_HappyCase(t *testing.T) {
	// Create and add config entry into GlobalAppCtx
	configEntryName := "ut-config"
	rkentry.RegisterConfigEntry(rkentry.WithNameConfig(configEntryName))
	defer rkentry.GlobalAppCtx.RemoveConfigEntry(configEntryName)

	resp := doConfigs(nil)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Entries, 1)
	assert.Equal(t, configEntryName, resp.Entries[0].EntryName)
}

func TestCommonServiceEntry_Configs_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	resp, err := entry.Configs(context.TODO(), &api.ConfigsRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_getSwUrl_HappyCase(t *testing.T) {
	// Create GwEntry with tls and sw enabled
	gwEntry := NewGwEntry()
	// Enable SwEntry
	gwEntry.SwEntry = NewSwEntry(
		WithPortSw(8080),
		WithPathSw("sw"))
	// Enable CertEntry with CertStore and ServerCert
	gwEntry.CertEntry = &rkentry.CertEntry{
		Store: &rkentry.CertStore{
			ServerCert: []byte("fake-cert"),
		},
	}

	res := getSwUrl(context.TODO(), gwEntry)
	assert.NotEmpty(t, res)
	assert.True(t, strings.HasPrefix(res, "https://"))
	assert.True(t, strings.HasSuffix(res, "8080/sw/"))
}

func TestCommonServiceEntry_getSwUrl_WithoutSw(t *testing.T) {
	// Create GwEntry with tls and sw enabled
	gwEntry := NewGwEntry()

	res := getSwUrl(context.TODO(), gwEntry)
	assert.Empty(t, res)
}

func TestCommonServiceEntry_getGwMapping_WithoutGw(t *testing.T) {
	grpcEntry := &GrpcEntry{}

	res := getGwMapping(context.TODO(), grpcEntry, "fake-grpc-method")
	assert.NotNil(t, res)
}

func TestCommonServiceEntry_getGwMapping_WithoutGwMapping(t *testing.T) {
	grpcEntry := &GrpcEntry{
		GwEntry: NewGwEntry(),
	}

	res := getGwMapping(context.TODO(), grpcEntry, "fake-grpc-method")
	assert.NotNil(t, res)
}

func TestCommonServiceEntry_getGwMapping_HappyCase(t *testing.T) {
	grpcEntry := &GrpcEntry{
		GwEntry: NewGwEntry(WithHttpPortGw(8080)),
	}

	grpcMethod := "fake-grpc-method"
	gwMethod := "fake-gw-method"
	gwPattern := "fake-gw-pattern"
	grpcEntry.GwEntry.GwMapping[grpcMethod] = &gwRule{
		Method:  gwMethod,
		Pattern: gwPattern,
	}

	grpcEntry.GwEntry.SwEntry = NewSwEntry(
		WithPortSw(8080),
		WithPathSw("sw"))

	// Enable CertEntry with CertStore and ServerCert
	grpcEntry.GwEntry.CertEntry = &rkentry.CertEntry{
		Store: &rkentry.CertStore{
			ServerCert: []byte("fake-cert"),
		},
	}

	res := getGwMapping(context.TODO(), grpcEntry, grpcMethod)
	assert.NotNil(t, res)
	assert.Equal(t, 8080, int(res.Port))
	assert.Equal(t, gwMethod, res.Method)
	assert.Equal(t, gwPattern, res.Pattern)
	assert.True(t, strings.HasPrefix(res.SwUrl, "https://"))
	assert.True(t, strings.HasSuffix(res.SwUrl, "8080/sw/"))
}

func TestCommonServiceEntry_doApis_WithoutGrpcEntry(t *testing.T) {
	res := doApis(context.TODO())

	assert.NotNil(t, res)
	assert.Empty(t, res.Entries)
}

func TestCommonServiceEntry_Apis_WithoutGrpcMethods(t *testing.T) {
	entry := NewCommonServiceEntry()

	resp, err := entry.Apis(context.TODO(), &api.ApisRequest{})

	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_Apis_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: create grpc entry
	grpcEntry := RegisterGrpcEntry()
	grpcEntry.Server = grpc.NewServer()
	// 2: register common service into grpc entry
	registerRkCommonService(grpcEntry.Server)

	// 3: create context with entry name into it
	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, grpcEntry.GetName())

	// 4: call function, now we can find common service methods from grpc server
	resp, err := entry.Apis(ctx, &api.ApisRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)

	rkentry.GlobalAppCtx.RemoveEntry(grpcEntry.GetName())
}

func TestCommonServiceEntry_Req_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: create grpc entry
	grpcEntry := RegisterGrpcEntry()
	grpcEntry.Server = grpc.NewServer()
	// 2: register common service into grpc entry
	registerRkCommonService(grpcEntry.Server)

	// 3: create context with entry name and rpc type into it
	ctx := rkgrpcinter.WrapContextForServer(context.TODO())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, grpcEntry.GetName())
	rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)

	// 4: we need to add prom metrics
	rkgrpcmetrics.UnaryServerInterceptor(
		rkgrpcmetrics.WithEntryNameAndType(grpcEntry.GetName(), grpcEntry.GetType()))

	// 5: call function, now we can find common service methods from grpc server
	resp, err := entry.Req(ctx, &api.ReqRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	fmt.Println(resp)

	rkentry.GlobalAppCtx.RemoveEntry(grpcEntry.GetName())
}

func TestCommonServiceEntry_doSys_HappyCase(t *testing.T) {
	res := doSys(context.TODO())

	assert.NotNil(t, res)
	assert.NotNil(t, res.CpuInfo)
	assert.NotNil(t, res.MemInfo)
	assert.NotNil(t, res.NetInfo)
	assert.NotNil(t, res.OsInfo)
	assert.NotNil(t, res.GoEnvInfo)
}

func TestCommonServiceEntry_Sys_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	res, err := entry.Sys(context.TODO(), &api.SysRequest{})

	assert.Nil(t, err)
	assert.NotNil(t, res)

	resAsMap := res.AsMap()
	assert.NotNil(t, resAsMap["cpuInfo"])
	assert.NotNil(t, resAsMap["memInfo"])
	assert.NotNil(t, resAsMap["netInfo"])
	assert.NotNil(t, resAsMap["osInfo"])
	assert.NotNil(t, resAsMap["goEnvInfo"])
}

func TestCommonServiceEntry_doEntriesHelper_HappyCase(t *testing.T) {
	// Make map of entry whose k/v pair is entryType.Entry
	fakeEntry := &FakeEntry{}
	m := map[string]rkentry.Entry{
		"fake-entry": fakeEntry,
	}

	res := &rkentry.EntriesResponse{
		Entries: make(map[string][]*rkentry.EntriesResponse_Entry),
	}

	doEntriesHelper(m, res)

	assert.NotEmpty(t, res.Entries)

	entries := res.Entries[fakeEntry.GetType()]
	assert.NotEmpty(t, entries)
	assert.Equal(t, fakeEntry.GetName(), entries[0].EntryName)
	assert.Equal(t, fakeEntry.GetType(), entries[0].EntryType)
	assert.Equal(t, fakeEntry.GetDescription(), entries[0].EntryDescription)
	assert.Equal(t, fakeEntry, entries[0].EntryMeta)
}

func TestCommonServiceEntry_doEntries_HappyCase(t *testing.T) {
	// Make map of entry whose k/v pair is entryType.Entry
	fakeEntry := &FakeEntry{}
	rkentry.GlobalAppCtx.AddEntry(fakeEntry)
	defer rkentry.GlobalAppCtx.RemoveEntry(fakeEntry.GetName())

	res := doEntries(context.TODO())

	assert.NotEmpty(t, res.Entries)

	entries := res.Entries[fakeEntry.GetType()]
	assert.NotEmpty(t, entries)
	assert.Equal(t, fakeEntry.GetName(), entries[0].EntryName)
	assert.Equal(t, fakeEntry.GetType(), entries[0].EntryType)
	assert.Equal(t, fakeEntry.GetDescription(), entries[0].EntryDescription)
	assert.Equal(t, fakeEntry, entries[0].EntryMeta)
}

func TestCommonServiceEntry_Entries_HappyCase(t *testing.T) {
	// Make map of entry whose k/v pair is entryType.Entry
	fakeEntry := &FakeEntry{}
	rkentry.GlobalAppCtx.AddEntry(fakeEntry)
	defer rkentry.GlobalAppCtx.RemoveEntry(fakeEntry.GetName())

	entry := NewCommonServiceEntry()

	resp, err := entry.Entries(context.TODO(), &api.EntriesRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_doCerts_HappyCase(t *testing.T) {
	certEntry := &rkentry.CertEntry{
		EntryName:        "fake-entry",
		EntryType:        "fake-type",
		EntryDescription: "fake-description",
		ServerCertPath:   "fake-server-cert-path",
		ServerKeyPath:    "fake-server-key-path",
		ClientCertPath:   "fake-client-cert-path",
		ClientKeyPath:    "fake-client-key-path",
		Retriever: &rkentry.CredRetrieverLocalFs{
			Locale:   "fake-locale",
			Provider: "fake-provider",
		},
		Store: &rkentry.CertStore{},
	}

	rkentry.GlobalAppCtx.AddCertEntry(certEntry)
	defer rkentry.GlobalAppCtx.RemoveCertEntry(certEntry.GetName())

	res := doCerts(context.TODO())
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.Entries)

	assert.Equal(t, "fake-entry", res.Entries[0].EntryName)
	assert.Equal(t, "fake-type", res.Entries[0].EntryType)
	assert.Equal(t, "fake-description", res.Entries[0].EntryDescription)
	assert.Equal(t, "fake-locale", res.Entries[0].Locale)
	assert.Equal(t, "fake-provider", res.Entries[0].Provider)
	assert.Equal(t, "fake-server-cert-path", res.Entries[0].ServerCertPath)
	assert.Equal(t, "fake-server-key-path", res.Entries[0].ServerKeyPath)
	assert.Equal(t, "fake-client-cert-path", res.Entries[0].ClientCertPath)
	assert.Equal(t, "fake-client-key-path", res.Entries[0].ClientKeyPath)
	assert.Empty(t, res.Entries[0].ServerCert)
	assert.Empty(t, res.Entries[0].ClientCert)
}

func TestCommonServiceEntry_Certs_HappyCase(t *testing.T) {
	certEntry := &rkentry.CertEntry{}

	rkentry.GlobalAppCtx.AddCertEntry(certEntry)
	defer rkentry.GlobalAppCtx.RemoveCertEntry(certEntry.GetName())

	entry := NewCommonServiceEntry()

	resp, err := entry.Certs(context.TODO(), &api.CertsRequest{})

	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_doLogsHelper_HappyCase(t *testing.T) {
	// Make map of entry whose k/v pair is entryType.Entry
	defaultZapLoggerConfig := rklogger.NewZapStdoutConfig()
	defaultZapLogger, _ := defaultZapLoggerConfig.Build()

	zapLogEntry := &rkentry.ZapLoggerEntry{
		EntryName:        "fake-name",
		EntryType:        "fake-type",
		EntryDescription: "fake-description",
		Logger:           defaultZapLogger,
		LoggerConfig:     defaultZapLoggerConfig,
	}

	m := map[string]rkentry.Entry{
		zapLogEntry.GetType(): zapLogEntry,
	}

	res := &rkentry.LogsResponse{
		Entries: make(map[string][]*rkentry.LogsResponse_Entry),
	}

	doLogsHelper(m, res)

	assert.NotEmpty(t, res.Entries)

	entries := res.Entries[zapLogEntry.GetType()]
	assert.NotEmpty(t, entries)
	assert.Equal(t, zapLogEntry.GetName(), entries[0].EntryName)
	assert.Equal(t, zapLogEntry.GetType(), entries[0].EntryType)
	assert.Equal(t, zapLogEntry.GetDescription(), entries[0].EntryDescription)
	assert.Equal(t, zapLogEntry, entries[0].EntryMeta)
	assert.Contains(t, entries[0].OutputPaths, "stdout")
	assert.Contains(t, entries[0].ErrorOutputPaths, "stderr")
}

func TestCommonServiceEntry_doLogs_HappyCase(t *testing.T) {
	// Make map of entry whose k/v pair is entryType.Entry
	defaultZapLoggerConfig := rklogger.NewZapStdoutConfig()
	defaultZapLogger, _ := defaultZapLoggerConfig.Build()

	zapLogEntry := &rkentry.ZapLoggerEntry{
		EntryName:        "fake-name",
		EntryType:        rkentry.ZapLoggerEntryType,
		EntryDescription: "fake-description",
		Logger:           defaultZapLogger,
		LoggerConfig:     defaultZapLoggerConfig,
	}

	rkentry.GlobalAppCtx.AddZapLoggerEntry(zapLogEntry)
	defer rkentry.GlobalAppCtx.RemoveZapLoggerEntry(zapLogEntry.GetName())

	res := doLogs(context.TODO())

	assert.NotEmpty(t, res.Entries)

	entries := res.Entries[zapLogEntry.GetType()]
	assert.NotEmpty(t, entries)
	assert.Len(t, entries, 2)
}

func TestCommonServiceEntry_Logs_HappyCase(t *testing.T) {
	// Make map of entry whose k/v pair is entryType.Entry
	defaultZapLoggerConfig := rklogger.NewZapStdoutConfig()
	defaultZapLogger, _ := defaultZapLoggerConfig.Build()

	zapLogEntry := &rkentry.ZapLoggerEntry{
		EntryName:        "fake-name",
		EntryType:        "fake-type",
		EntryDescription: "fake-description",
		Logger:           defaultZapLogger,
		LoggerConfig:     defaultZapLoggerConfig,
	}

	rkentry.GlobalAppCtx.AddZapLoggerEntry(zapLogEntry)
	defer rkentry.GlobalAppCtx.RemoveZapLoggerEntry(zapLogEntry.GetName())

	entry := NewCommonServiceEntry()

	resp, err := entry.Logs(context.TODO(), &api.LogsRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_Git_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: add rk meta entry into GlobalAppCtx
	rkentry.GlobalAppCtx.SetRkMetaEntry(&rkentry.RkMetaEntry{
		RkMeta: &rkcommon.RkMeta{
			Git: &rkcommon.Git{
				Commit: &rkcommon.Commit{
					Committer: &rkcommon.Committer{},
				},
			},
		},
	})

	// 2: call function, now we can find common service methods from grpc server
	resp, err := entry.Git(context.TODO(), &api.GitRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_Dep_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: call function, now we can find common service methods from grpc server
	resp, err := entry.Deps(context.TODO(), &api.DepsRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_Readme_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: call function, now we can find common service methods from grpc server
	resp, err := entry.Readme(context.TODO(), &api.ReadmeRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_License_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: call function, now we can find common service methods from grpc server
	resp, err := entry.License(context.TODO(), &api.LicenseRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestCommonServiceEntry_GwErrorMapping_HappyCase(t *testing.T) {
	entry := NewCommonServiceEntry()

	// 1: call function, now we can find common service methods from grpc server
	resp, err := entry.GwErrorMapping(context.TODO(), &api.GwErrorMappingRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func assertPanic(t *testing.T) {
	if r := recover(); r != nil {
		// expect panic to be called with non nil error
		assert.True(t, true)
	} else {
		// this should never be called in case of a bug
		assert.True(t, false)
	}
}

func assertNotPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, false)
	} else {
		// This should never be called in case of a bug
		assert.True(t, true)
	}
}

type FakeEntry struct{}

func (entry *FakeEntry) GetName() string {
	return "fake-name"
}

func (entry *FakeEntry) GetType() string {
	return "fake-type"
}

func (entry *FakeEntry) GetDescription() string {
	return "fake-description"
}

func (entry *FakeEntry) String() string {
	return "string"
}

func (entry *FakeEntry) Bootstrap(context.Context) {}

func (entry *FakeEntry) Interrupt(context.Context) {}
