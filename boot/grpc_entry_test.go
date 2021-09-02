// +build !race

// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"
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
	gwEntry := NewGwEntry()
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
		WithGwEntryGrpc(gwEntry),
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
	assert.Equal(t, gwEntry, entry.GwEntry)
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
cert:                                         # Optional
  - name: "local-cert"                        # Required
    description: "Description of entry"       # Optional
    provider: "localFs"                       # Required, etcd, consul, localFs, remoteFs are supported options
    locale: "*::*::*::*"                      # Optional, default: *::*::*::*
    serverCertPath: "example/boot/full/server.pem"      # Optional, default: "", path of certificate on local FS
    serverKeyPath: "example/boot/full/server-key.pem"   # Optional, default: "", path of certificate on local FS
grpc:
  - name: greeter
    port: 1949
    commonService:
      enabled: true                                  # Optional, default: false
    cert:
      ref: "local-cert"                              # Optional, default: "", reference of cert entry declared above
    gw:
      enabled: true
      port: 8080
      cert:
        ref: "local-cert"
      pathPrefix: "/rk/v1/"                          # Optional, default: "/rk/v1/"
      tv:
        enabled: true                                 # Optional, default: false
      sw:
        enabled: true                                  # Optional, default: false
        path: "sw"                                     # Optional, default: "sw"
        headers: [ "sw:rk" ]                             # Optional, default: []
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
	assert.Equal(t, "local-cert", entry.CertEntry.GetName())
	assert.Equal(t, "greeter", entry.GetName())
	assert.Equal(t, uint64(1949), entry.Port)
	assert.NotNil(t, entry.CommonServiceEntry)
	assert.NotNil(t, entry.GwEntry)

	assert.Len(t, entry.UnaryInterceptors, 4)
	assert.Len(t, entry.StreamInterceptors, 4)
}

func createFileAtTestTempDir(t *testing.T, content string) string {
	tempDir := path.Join(t.TempDir(), "ut-boot.yaml")
	assert.Nil(t, ioutil.WriteFile(tempDir, []byte(content), os.ModePerm))
	return tempDir
}
