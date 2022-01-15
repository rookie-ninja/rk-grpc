// +build !race

// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-entry/middleware/cors"
	"github.com/rookie-ninja/rk-entry/middleware/csrf"
	rkmidmetrics "github.com/rookie-ninja/rk-entry/middleware/metrics"
	"github.com/rookie-ninja/rk-entry/middleware/secure"
	testdata "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/testdata"
	"github.com/rookie-ninja/rk-grpc/interceptor/meta"
	rkgrpcmetrics "github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"
	"time"
)

//func TestRegisterGrpcEntry_WithoutOptions(t *testing.T) {
//	entry := RegisterGrpcEntry()
//
//	assert.NotNil(t, entry)
//
//	entryDefaultName := "GrpcServer-" + strconv.FormatUint(entry.Port, 10)
//	assert.Equal(t, entry, rkentry.GlobalAppCtx.GetEntry(entryDefaultName))
//	assert.True(t, rkentry.GlobalAppCtx.RemoveEntry(entryDefaultName))
//
//	assert.Len(t, entry.UnaryInterceptors, 1)
//	assert.Len(t, entry.StreamInterceptors, 1)
//	assert.NotNil(t, entry.ZapLoggerEntry)
//	assert.NotNil(t, entry.EventLoggerEntry)
//}
//
//func TestRegisterGrpcEntry_HappyCase(t *testing.T) {
//	entryName := "ut-grpc-entry"
//	zapLoggerEntry := rkentry.NoopZapLoggerEntry()
//	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
//	grpcPort := uint64(2020)
//	serverOpt := grpc.InitialWindowSize(1)
//	loggingInterUnary := rkgrpclog.UnaryServerInterceptor()
//	loggingInterStream := rkgrpclog.StreamServerInterceptor()
//	certEntry := &rkentry.CertEntry{}
//	commonServiceEntry := RegisterCommonServiceEntry()
//
//	entry := RegisterGrpcEntry(
//		WithNameGrpc(entryName),
//		WithZapLoggerEntryGrpc(zapLoggerEntry),
//		WithEventLoggerEntryGrpc(eventLoggerEntry),
//		WithPortGrpc(grpcPort),
//		WithServerOptionsGrpc(serverOpt),
//		WithUnaryInterceptorsGrpc(loggingInterUnary),
//		WithStreamInterceptorsGrpc(loggingInterStream),
//		WithCertEntryGrpc(certEntry),
//		WithCommonServiceEntryGrpc(commonServiceEntry))
//
//	assert.NotNil(t, entry)
//
//	assert.Equal(t, entry, rkentry.GlobalAppCtx.GetEntry(entryName))
//	assert.Equal(t, zapLoggerEntry, entry.ZapLoggerEntry)
//	assert.Equal(t, eventLoggerEntry, entry.EventLoggerEntry)
//	assert.Equal(t, grpcPort, entry.Port)
//	assert.Len(t, entry.ServerOpts, 1)
//	assert.Len(t, entry.UnaryInterceptors, 2)
//	assert.Len(t, entry.StreamInterceptors, 2)
//	assert.Equal(t, certEntry, entry.CertEntry)
//	assert.Equal(t, commonServiceEntry, entry.CommonServiceEntry)
//
//	assert.True(t, rkentry.GlobalAppCtx.RemoveEntry(entryName))
//}
//
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
  proxy:
    enabled: true
    rules:
      - type: headerBased
        headerPairs: ["domain:test"]
        dest: ["localhost:8081"]
      - type: pathBased
        paths: [""]
        dest: [""]
      - type: IpBased
        Ips: [""]
        dest: [""]
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
    jwt:
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
	entry.Bootstrap(context.TODO())

	bytes, err := entry.MarshalJSON()
	assert.NotEmpty(t, bytes)
	assert.Nil(t, err)

	time.Sleep(time.Second)
	// endpoint should be accessible with 8080 port
	validateServerIsUp(t, entry.Port)

	entry.Interrupt(context.Background())
}

func TestRegisterGrpcEntry(t *testing.T) {
	// without options
	entry := RegisterGrpcEntry()
	assert.NotNil(t, entry)
	assert.False(t, entry.IsTlsEnabled())
	assert.False(t, entry.IsCommonServiceEnabled())
	assert.False(t, entry.IsProxyEnabled())
	assert.False(t, entry.IsSwEnabled())
	assert.False(t, entry.IsStaticFileHandlerEnabled())
	assert.False(t, entry.IsTvEnabled())
	assert.False(t, entry.IsPromEnabled())

	// with options, invalid tls
	entry = RegisterGrpcEntry(
		WithName("name"),
		WithDescription("desc"),
		WithZapLoggerEntry(nil),
		WithEventLoggerEntry(nil),
		WithPort(8080),
		WithServerOptions(grpc.MaxRecvMsgSize(10)),
		WithUnaryInterceptors(rkgrpcmeta.UnaryServerInterceptor()),
		WithStreamInterceptors(rkgrpcmeta.StreamServerInterceptor()),
		WithGrpcRegF(func(server *grpc.Server) {}),
		WithCertEntry(rkentry.RegisterCertEntry()),
		WithCommonServiceEntry(rkentry.RegisterCommonServiceEntry()),
		WithEnableReflection(true),
		WithSwEntry(rkentry.RegisterSwEntry()),
		WithTvEntry(rkentry.RegisterTvEntry()),
		WithProxyEntry(NewProxyEntry()),
		WithPromEntry(rkentry.RegisterPromEntry()),
		WithStaticFileHandlerEntry(rkentry.RegisterStaticFileHandlerEntry()),
		WithGwRegF(func(ctx context.Context, mux *gwruntime.ServeMux, s string, options []grpc.DialOption) error {
			return nil
		}),
		WithGrpcDialOptions(grpc.WithBlock()),
		WithGwMuxOptions(gwruntime.WithDisablePathLengthFallback()),
		WithGwMappingFilePaths(""),
	)
	assert.True(t, entry.IsCommonServiceEnabled())
	assert.True(t, entry.IsProxyEnabled())
	assert.True(t, entry.IsSwEnabled())
	assert.True(t, entry.IsStaticFileHandlerEnabled())
	assert.True(t, entry.IsTvEnabled())
	assert.True(t, entry.IsPromEnabled())
	assert.NotNil(t, entry)

	// with valid tls
	certEntry := rkentry.RegisterCertEntry()
	certEntry.Store.ServerCert, certEntry.Store.ServerKey = generateCerts()
	entry = RegisterGrpcEntry(
		WithName("name"),
		WithPort(8080),
		WithCertEntry(certEntry),
	)

	assert.True(t, entry.IsTlsEnabled())
	assert.NotNil(t, entry)
}

func TestRegisterGrpcEntry_PublicFunc(t *testing.T) {
	serverOpt := grpc.MaxRecvMsgSize(10)
	unaryInter := rkgrpcmeta.UnaryServerInterceptor()
	streamInter := rkgrpcmeta.StreamServerInterceptor()
	corsOpt := rkmidcors.WithEntryNameAndType("", "")
	csrfOpt := rkmidcsrf.WithEntryNameAndType("", "")
	secOpt := rkmidsec.WithEntryNameAndType("", "")
	regFuncGrpc := func(server *grpc.Server) {}
	regFuncGw := func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error { return nil }
	gwDialOpt := grpc.WithBlock()

	entry := RegisterGrpcEntry()

	entry.AddServerOptions(serverOpt)
	entry.AddUnaryInterceptors(unaryInter)
	entry.AddStreamInterceptors(streamInter)
	entry.AddGwCorsOptions(corsOpt)
	entry.AddGwCsrfOptions(csrfOpt)
	entry.AddGwSecureOptions(secOpt)
	entry.AddRegFuncGrpc(regFuncGrpc)
	entry.AddRegFuncGw(regFuncGw)
	entry.AddGwDialOptions(gwDialOpt)

	assert.NotEmpty(t, entry.ServerOpts)
	assert.NotEmpty(t, entry.UnaryInterceptors)
	assert.NotEmpty(t, entry.StreamInterceptors)
	assert.NotEmpty(t, entry.gwCorsOptions)
	assert.NotEmpty(t, entry.gwCsrfOptions)
	assert.NotEmpty(t, entry.gwSecureOptions)
	assert.NotEmpty(t, entry.GrpcRegF)
	assert.NotEmpty(t, entry.GwRegF)
	assert.NotEmpty(t, entry.GwDialOptions)

	bytes, err := entry.MarshalJSON()
	assert.NotEmpty(t, bytes)
	assert.Nil(t, err)

	assert.Nil(t, entry.UnmarshalJSON([]byte{}))
}

func TestRegisterGrpcEntry_EntryFunc(t *testing.T) {
	defer assertNotPanic(t)

	// case without options
	entry := RegisterGrpcEntry()
	entry.Bootstrap(context.TODO())
	time.Sleep(1 * time.Second)
	entry.Interrupt(context.TODO())
	assert.NotEmpty(t, entry.GetName())
	assert.NotEmpty(t, entry.GetType())
	assert.NotEmpty(t, entry.GetDescription())
	assert.NotEmpty(t, entry.String())

	// case 2
	entry = RegisterGrpcEntry(
		WithUnaryInterceptors(rkgrpcmeta.UnaryServerInterceptor()),
		WithStreamInterceptors(rkgrpcmeta.StreamServerInterceptor()))
	entry.Bootstrap(context.TODO())
	time.Sleep(1 * time.Second)
	entry.Interrupt(context.TODO())

	// case 3
	entry = RegisterGrpcEntry(
		WithProxyEntry(NewProxyEntry()))
	entry.Bootstrap(context.TODO())
	time.Sleep(1 * time.Second)
	entry.Interrupt(context.TODO())

	// case 4, 5, 6, 7, 8
	certEntry := rkentry.RegisterCertEntry()
	certEntry.Store.ServerCert, certEntry.Store.ServerKey = generateCerts()
	entry = RegisterGrpcEntry(
		WithServerOptions(grpc.MaxRecvMsgSize(10)),
		WithCertEntry(certEntry))
	entry.Bootstrap(context.TODO())
	assert.NotEmpty(t, entry.String())
	time.Sleep(1 * time.Second)
	entry.Interrupt(context.TODO())

	// case 9
	entry = RegisterGrpcEntry()
	entry.AddRegFuncGw(func(ctx context.Context, mux *gwruntime.ServeMux, s string, options []grpc.DialOption) error {
		return nil
	})
	entry.Bootstrap(context.TODO())
	time.Sleep(1 * time.Second)
	entry.Interrupt(context.TODO())

	// case 11, 12, 13, 14, 15, 16, 17, 18, 19
	entry = RegisterGrpcEntry(
		WithSwEntry(rkentry.RegisterSwEntry()),
		WithTvEntry(rkentry.RegisterTvEntry()),
		WithStaticFileHandlerEntry(rkentry.RegisterStaticFileHandlerEntry()),
		WithPromEntry(rkentry.RegisterPromEntry()),
		WithCommonServiceEntry(rkentry.RegisterCommonServiceEntry()))
	corsOpt := rkmidcors.WithEntryNameAndType("", "")
	csrfOpt := rkmidcsrf.WithEntryNameAndType("", "")
	secOpt := rkmidsec.WithEntryNameAndType("", "")
	entry.AddGwCorsOptions(corsOpt)
	entry.AddGwSecureOptions(secOpt)
	entry.AddGwCsrfOptions(csrfOpt)

	entry.Bootstrap(context.TODO())
	time.Sleep(1 * time.Second)
	entry.Interrupt(context.TODO())
}

func TestGrpcEntry_startGrpcServer_Panic(t *testing.T) {
	// without stopped error
	defer assertPanic(t)
	entry := RegisterGrpcEntry()
	logger, _ := zap.NewDevelopment()
	lis := &ErrListener{}
	entry.Server = grpc.NewServer()
	entry.startGrpcServer(lis, logger)

	defer lis.Close()
}

func TestGrpcEntry_startHttpServer_Panic(t *testing.T) {
	defer assertPanic(t)

	// happy case
	entry := RegisterGrpcEntry()
	entry.HttpServer = &http.Server{}
	lis := &ErrListener{}
	logger, _ := zap.NewDevelopment()
	entry.startHttpServer(lis, logger)
	defer lis.Close()
}

func TestGetGrpcEntry_parseGwMapping(t *testing.T) {
	entry := RegisterGrpcEntry()

	// case 1: failed to read file
	entry.GwMappingFilePaths = []string{""}
	entry.parseGwMapping()
	assert.Empty(t, entry.GwHttpToGrpcMapping)

	// case 2: invalid yaml format
	invalidYamlStr := `invalid yaml`
	entry.GwMappingFilePaths = []string{createFileAtTestTempDir(t, invalidYamlStr)}
	entry.parseGwMapping()
	assert.Empty(t, entry.GwHttpToGrpcMapping)

	// case 3: failed to unmarshal
	validYamlStr := `
---
key: value
`
	entry.GwMappingFilePaths = []string{createFileAtTestTempDir(t, validYamlStr)}
	entry.parseGwMapping()
	entry.parseGwMapping()
	assert.Empty(t, entry.GwHttpToGrpcMapping)

	// case 4: happy case
	validYamlStr = `
---
type: google.api.Service
config_version: 3
http:
  rules:
    - selector: api.v1.Greeter.Get
      get: /v1/get
    - selector: api.v1.Greeter.Put
      put: /v1/put
    - selector: api.v1.Greeter.Post
      post: /v1/post
    - selector: api.v1.Greeter.Delete
      delete: /v1/delete
    - selector: api.v1.Greeter.Patch
      patch: /v1/patch
`
	entry.GwMappingFilePaths = []string{createFileAtTestTempDir(t, validYamlStr)}
	entry.parseGwMapping()
	assert.Equal(t, "/v1/get", entry.GwHttpToGrpcMapping["api.v1.Greeter.Get"].Pattern)
	assert.Equal(t, "GET", entry.GwHttpToGrpcMapping["api.v1.Greeter.Get"].Method)

	assert.Equal(t, "/v1/put", entry.GwHttpToGrpcMapping["api.v1.Greeter.Put"].Pattern)
	assert.Equal(t, "PUT", entry.GwHttpToGrpcMapping["api.v1.Greeter.Put"].Method)

	assert.Equal(t, "/v1/post", entry.GwHttpToGrpcMapping["api.v1.Greeter.Post"].Pattern)
	assert.Equal(t, "POST", entry.GwHttpToGrpcMapping["api.v1.Greeter.Post"].Method)

	assert.Equal(t, "/v1/delete", entry.GwHttpToGrpcMapping["api.v1.Greeter.Delete"].Pattern)
	assert.Equal(t, "DELETE", entry.GwHttpToGrpcMapping["api.v1.Greeter.Delete"].Method)

	assert.Equal(t, "/v1/patch", entry.GwHttpToGrpcMapping["api.v1.Greeter.Patch"].Pattern)
	assert.Equal(t, "PATCH", entry.GwHttpToGrpcMapping["api.v1.Greeter.Patch"].Method)

	assert.NotEmpty(t, entry.GwHttpToGrpcMapping)

}

func TestGetGrpcEntry(t *testing.T) {
	RegisterGrpcEntry(WithName("ut"))

	assert.NotNil(t, GetGrpcEntry("ut"))
	assert.Nil(t, GetGrpcEntry("not-exist"))
}

func TestGrpcEntry_Apis(t *testing.T) {
	entry := RegisterGrpcEntry()
	entry.Server = grpc.NewServer()

	// without apis
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ut", nil)
	entry.Apis(w, req)

	apisResponse := &rkentry.ApisResponse{}
	json.Unmarshal(w.Body.Bytes(), apisResponse)
	assert.Empty(t, apisResponse.Entries)

	// with apis
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ut", nil)
	testdata.RegisterGreeterServer(entry.Server, &GreeterServer{})
	entry.Apis(w, req)

	apisResponse = &rkentry.ApisResponse{}
	json.Unmarshal(w.Body.Bytes(), apisResponse)
	assert.NotEmpty(t, apisResponse.Entries)
}

func TestGrpcEntry_Req(t *testing.T) {
	// without metrics set
	entry := RegisterGrpcEntry()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ut", nil)
	entry.Req(w, req)

	reqResponse := &rkentry.ReqResponse{}
	json.Unmarshal(w.Body.Bytes(), reqResponse)
	assert.Empty(t, reqResponse.Metrics)

	// with metrics
	entry = RegisterGrpcEntry(
		WithName("ut"),
		WithPromEntry(rkentry.RegisterPromEntry()),
		WithUnaryInterceptors(rkgrpcmetrics.UnaryServerInterceptor(
			rkmidmetrics.WithEntryNameAndType("ut", GrpcEntryType))))
	entry.Server = grpc.NewServer()
	testdata.RegisterGreeterServer(entry.Server, &GreeterServer{})

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ut", nil)
	entry.Req(w, req)

	reqResponse = &rkentry.ReqResponse{}
	json.Unmarshal(w.Body.Bytes(), reqResponse)
	assert.NotEmpty(t, reqResponse.Metrics)

	// with metrics from tv
	entry = RegisterGrpcEntry(
		WithName("ut"),
		WithPromEntry(rkentry.RegisterPromEntry()),
		WithUnaryInterceptors(rkgrpcmetrics.UnaryServerInterceptor(
			rkmidmetrics.WithEntryNameAndType("ut", GrpcEntryType))))
	entry.Server = grpc.NewServer()
	testdata.RegisterGreeterServer(entry.Server, &GreeterServer{})

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ut", nil)
	req.URL.RawQuery = "fromTv=true"
	entry.Req(w, req)

	reqResponse = &rkentry.ReqResponse{}
	json.Unmarshal(w.Body.Bytes(), reqResponse)
	assert.NotEmpty(t, reqResponse.Metrics)
	assert.Equal(t, reqResponse.Metrics[0].GrpcService, reqResponse.Metrics[0].RestMethod)
	assert.Equal(t, reqResponse.Metrics[0].GrpcMethod, reqResponse.Metrics[0].RestPath)
}

func TestGrpcEntry_GwErrorMapping(t *testing.T) {
	entry := RegisterGrpcEntry()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ut", nil)
	entry.GwErrorMapping(w, req)

	res := &rkentry.GwErrorMappingResponse{}
	json.Unmarshal(w.Body.Bytes(), res)
	assert.NotEmpty(t, res.Mapping)
}

func TestGrpcEntry_TV(t *testing.T) {
	// happy case
	entry := RegisterGrpcEntry()
	tvEntry := rkentry.RegisterTvEntry()
	tvEntry.Bootstrap(context.TODO())
	entry.TvEntry = tvEntry

	entry.Server = grpc.NewServer()
	testdata.RegisterGreeterServer(entry.Server, &GreeterServer{})

	// 1: /apis
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rk/v1/tv/apis", nil)

	entry.TV(w, req)
	assert.NotEmpty(t, w.Body.String())

	// 2: /gwErrorMapping
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/rk/v1/tv/gwErrorMapping", nil)

	entry.TV(w, req)
	assert.NotEmpty(t, w.Body.String())

	// 3: /env
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/rk/v1/tv/env", nil)

	entry.TV(w, req)
	assert.NotEmpty(t, w.Body.String())
}

func TestGrpcEntry_getSwUrl(t *testing.T) {
	// without swagger entry enabled
	entry := RegisterGrpcEntry()
	assert.Empty(t, entry.getSwUrl(nil))

	// with SwEntry and CertEntry
	entry.SwEntry = rkentry.RegisterSwEntry()
	certEntry := rkentry.RegisterCertEntry()
	certEntry.Store.ServerCert, certEntry.Store.ServerKey = generateCerts()
	entry.CertEntry = certEntry
	assert.NotEmpty(t, entry.getSwUrl(httptest.NewRequest(http.MethodGet, "/", nil)))
}

// GreeterServer Implementation of GreeterServer.
type GreeterServer struct{}

// SayHello Handle SayHello method.
func (server *GreeterServer) SayHello(ctx context.Context, request *testdata.HelloRequest) (*testdata.HelloResponse, error) {
	return &testdata.HelloResponse{
		Message: fmt.Sprintf("Hello %s!", request.GetName()),
	}, nil
}

type ErrListener struct{}

func (e ErrListener) Accept() (net.Conn, error) {
	return nil, errors.New("")
}

func (e ErrListener) Close() error {
	return nil
}

func (e ErrListener) Addr() net.Addr {
	return &net.TCPAddr{}
}

func generateCerts() ([]byte, []byte) {
	// Create certs and return as []byte
	ca := &x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{"Fake cert."},
		},
		SerialNumber:          big.NewInt(42),
		NotAfter:              time.Now().Add(2 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Create a Private Key
	key, _ := rsa.GenerateKey(rand.Reader, 4096)

	// Use CA Cert to sign a CSR and create a Public Cert
	csr := &key.PublicKey
	cert, _ := x509.CreateCertificate(rand.Reader, ca, ca, csr, key)

	// Convert keys into pem.Block
	c := &pem.Block{Type: "CERTIFICATE", Bytes: cert}
	k := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}

	return pem.EncodeToMemory(c), pem.EncodeToMemory(k)
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
