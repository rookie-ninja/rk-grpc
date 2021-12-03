// +build !race

// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	//gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"testing"
	"time"
)

func TestRegisterGrpcEntry_WithoutOptions(t *testing.T) {
	entry := RegisterGrpcEntry()

	assert.NotNil(t, entry)

	entryDefaultName := "GrpcServer-" + strconv.FormatUint(entry.Port, 10)
	assert.Equal(t, entry, rkentry.GlobalAppCtx.GetEntry(entryDefaultName))
	assert.True(t, rkentry.GlobalAppCtx.RemoveEntry(entryDefaultName))

	assert.Len(t, entry.UnaryInterceptors, 1)
	assert.Len(t, entry.StreamInterceptors, 1)
	assert.NotNil(t, entry.ZapLoggerEntry)
	assert.NotNil(t, entry.EventLoggerEntry)
}

func TestRegisterGrpcEntry_HappyCase(t *testing.T) {
	entryName := "ut-grpc-entry"
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
	grpcPort := uint64(2020)
	serverOpt := grpc.InitialWindowSize(1)
	loggingInterUnary := rkgrpclog.UnaryServerInterceptor()
	loggingInterStream := rkgrpclog.StreamServerInterceptor()
	certEntry := &rkentry.CertEntry{}
	commonServiceEntry := NewCommonServiceEntry()

	entry := RegisterGrpcEntry(
		WithNameGrpc(entryName),
		WithZapLoggerEntryGrpc(zapLoggerEntry),
		WithEventLoggerEntryGrpc(eventLoggerEntry),
		WithPortGrpc(grpcPort),
		WithServerOptionsGrpc(serverOpt),
		WithUnaryInterceptorsGrpc(loggingInterUnary),
		WithStreamInterceptorsGrpc(loggingInterStream),
		WithCertEntryGrpc(certEntry),
		WithCommonServiceEntryGrpc(commonServiceEntry))

	assert.NotNil(t, entry)

	assert.Equal(t, entry, rkentry.GlobalAppCtx.GetEntry(entryName))
	assert.Equal(t, zapLoggerEntry, entry.ZapLoggerEntry)
	assert.Equal(t, eventLoggerEntry, entry.EventLoggerEntry)
	assert.Equal(t, grpcPort, entry.Port)
	assert.Len(t, entry.ServerOpts, 1)
	assert.Len(t, entry.UnaryInterceptors, 2)
	assert.Len(t, entry.StreamInterceptors, 2)
	assert.Equal(t, certEntry, entry.CertEntry)
	assert.Equal(t, commonServiceEntry, entry.CommonServiceEntry)

	assert.True(t, rkentry.GlobalAppCtx.RemoveEntry(entryName))
}

func TestRegisterGrpcEntriesWithConfig_HappyCase(t *testing.T) {
	defer assertNotPanic(t)

	configFile := `
---
zapLogger:
  - name: zap-logger
eventLogger:
  - name: event-logger
grpc:
  - name: greeter
    port: 1949
    enabled: true
    commonService:
      enabled: true                                  # Optional, default: false
    cert:
      ref: "local-cert"                              # Optional, default: "", reference of cert entry declared above
    noRecvMsgSizeLimit: true
    enableRkGwOption: true
    gwOption:
      marshal:
        multiline: true
        emitUnpopulated: true
        indent: "  "
        allowPartial: true
        useProtoNames: true
        useEnumNumbers: true
      unmarshal:
        allowPartial: true
        discardUnknown: true
    tv:
      enabled: true                                  # Optional, default: false
    sw:
      enabled: true                                  # Optional, default: false
      path: "sw"                                     # Optional, default: "sw"
      headers: [ "sw:rk" ]                           # Optional, default: []
    prom:
      enabled: true                                  # Optional, default: false
      path: "metrics"                                # Optional, default: ""
      pusher:
        enabled: false                               # Optional, default: false
        jobName: "greeter-pusher"                    # Required
        remoteAddress: "localhost:9091"              # Required
        basicAuth: "user:pass"                       # Optional, default: ""
        intervalMS: 1000                             # Optional, default: 1000
        cert:
          ref: "local-cert"                          # Optional, default: "", reference of cert entry declared above
    logger:
      zapLogger:
        ref: zap-logger                              # Optional, default: logger of STDOUT, reference of logger entry declared above
      eventLogger:
        ref: event-logger                            # Optional, default: logger of STDOUT, reference of logger entry declared above
    interceptors:
      loggingZap:
        enabled: true                                # Optional, default: false
      metricsProm:
        enabled: true                                # Optional, default: false
      auth:
        enabled: true                                # Optional, default: false
        basic:
          - "user:pass"                              # Optional, default: ""
      meta:
        enabled: true                                # Optional, default: false
      tracingTelemetry:
        enabled: true                                # Optional, default: false
        exporter:                                    # Optional, default will create a stdout exporter
          file:
            enabled: true                            # Optional, default: false
      rateLimit:
        enabled: true
        algorithm: leakyBucket
        reqPerSec: 1
        paths:
          - path: "ut-method"
            reqPerSec: 1
      timeout:
        enabled: true
      cors:
        enabled: true
      secure:
        enabled: true
      csrf:
        enabled: true
`

	// Create bootstrap config file at ut temp dir
	configFilePath := createFileAtTestTempDir(t, configFile)

	// Register internal entries
	rkentry.RegisterInternalEntriesFromConfig(configFilePath)

	// Register entries with config file
	entries := RegisterGrpcEntriesWithConfig(configFilePath)
	assert.Len(t, entries, 1)
	entry := entries["greeter"].(*GrpcEntry)

	assert.Equal(t, "zap-logger", entry.ZapLoggerEntry.GetName())
	assert.Equal(t, "event-logger", entry.EventLoggerEntry.GetName())
	assert.Equal(t, "greeter", entry.GetName())
	assert.Equal(t, uint64(1949), entry.Port)
	assert.NotNil(t, entry.CommonServiceEntry)
	assert.NotNil(t, entry.SwEntry)
	assert.NotNil(t, entry.TvEntry)
	assert.NotNil(t, entry.PromEntry)

	assert.True(t, len(entry.UnaryInterceptors) > 0)
	assert.True(t, len(entry.StreamInterceptors) > 0)

	// Bootstrap
	ctx := context.WithValue(context.Background(), bootstrapEventIdKey, "ut")
	entry.Bootstrap(ctx)

	bytes, err := entry.MarshalJSON()
	assert.NotEmpty(t, bytes)
	assert.Nil(t, err)

	time.Sleep(time.Second)
	// endpoint should be accessible with 8080 port
	validateServerIsUp(t, entry.Port)

	entry.Interrupt(context.Background())
}

func TestGrpcEntry_UnmarshalJSON(t *testing.T) {
	entry := RegisterGrpcEntry()
	assert.Nil(t, entry.UnmarshalJSON(nil))
}

func TestGrpcEntry_GetDescription(t *testing.T) {
	entry := RegisterGrpcEntry()
	assert.NotEmpty(t, entry.GetDescription())
}

func TestGrpcEntry_AddRegFuncGrpc(t *testing.T) {
	entry := RegisterGrpcEntry()
	entry.AddRegFuncGrpc(func(server *grpc.Server) {})
	assert.Len(t, entry.GrpcRegF, 1)
}

func TestWithGwRegFGrpc(t *testing.T) {
	entry := RegisterGrpcEntry()
	entry.AddRegFuncGw(func(ctx context.Context, mux *gwruntime.ServeMux, s string, options []grpc.DialOption) error {
		return nil
	})
	assert.Len(t, entry.GwRegF, 1)
}

func TestGrpcEntry_AddServerOptions(t *testing.T) {
	entry := RegisterGrpcEntry()
	entry.AddServerOptions(grpc.EmptyServerOption{})
	assert.Len(t, entry.ServerOpts, 1)
}

func TestGrpcEntry_String(t *testing.T) {
	entry := RegisterGrpcEntry()
	assert.NotEmpty(t, entry.String())
}

func validateServerIsUp(t *testing.T, port uint64) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("0.0.0.0", strconv.FormatUint(port, 10)), time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	if conn != nil {
		assert.Nil(t, conn.Close())
	}
}

func createFileAtTestTempDir(t *testing.T, content string) string {
	tempDir := path.Join(t.TempDir(), "ut-boot.yaml")
	assert.Nil(t, ioutil.WriteFile(tempDir, []byte(content), os.ModePerm))
	return tempDir
}
