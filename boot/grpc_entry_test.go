//go:build !race
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
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rookie-ninja/rk-entry/v2/entry"
	"github.com/rookie-ninja/rk-entry/v2/middleware/cors"
	"github.com/rookie-ninja/rk-entry/v2/middleware/csrf"
	"github.com/rookie-ninja/rk-entry/v2/middleware/secure"
	testdata "github.com/rookie-ninja/rk-grpc/v2/example/middleware/proto/testdata"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/meta"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"testing"
	"time"
)

func TestRegisterGrpcEntriesWithConfig_HappyCase(t *testing.T) {
	defer assertNotPanic(t)

	configFile := `
---
logger:
- name: zap-logger
event:
- name: event-logger
grpc:
- name: greeter
  port: 1949
  enabled: true
  commonService:
    enabled: true                                  # Optional, default: false
  certEntry: "local-cert"                          # Optional, default: "", reference of cert entry declared above
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
  docs:
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
  loggerEntry: zap-logger                          # Optional, default: logger of STDOUT, reference of logger entry declared above
  eventEntry: event-logger                         # Optional, default: logger of STDOUT, reference of logger entry declared above
  middleware:
    logging:
      enabled: true                                # Optional, default: false
    prom:
      enabled: true                                # Optional, default: false
    auth:
      enabled: true                                # Optional, default: false
      basic:
        - "user:pass"                              # Optional, default: ""
    meta:
      enabled: true                                # Optional, default: false
    trace:
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

	// Register internal entries
	rkentry.BootstrapPreloadEntryYAML([]byte(configFile))

	// Register entries with config file
	entries := RegisterGrpcEntryYAML([]byte(configFile))
	assert.Len(t, entries, 1)
	entry := entries["greeter"].(*GrpcEntry)

	assert.Equal(t, "zap-logger", entry.LoggerEntry.GetName())
	assert.Equal(t, "event-logger", entry.EventEntry.GetName())
	assert.Equal(t, "greeter", entry.GetName())
	assert.Equal(t, uint64(1949), entry.Port)
	assert.NotNil(t, entry.CommonServiceEntry)
	assert.NotNil(t, entry.SWEntry)
	assert.NotNil(t, entry.DocsEntry)
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
	assert.False(t, entry.IsSWEnabled())
	assert.False(t, entry.IsStaticFileHandlerEnabled())
	assert.False(t, entry.IsDocsEnabled())
	assert.False(t, entry.IsPromEnabled())

	certEntry := rkentry.RegisterCertEntry(&rkentry.BootCert{
		Cert: []*rkentry.BootCertE{
			{
				Name: "ut-cert",
			},
		},
	})[0]
	certificate, _ := tls.X509KeyPair(generateCerts())
	certEntry.Certificate = &certificate

	// with options, invalid tls
	entry = RegisterGrpcEntry(
		WithName("name"),
		WithDescription("desc"),
		WithLoggerEntry(rkentry.LoggerEntryNoop),
		WithEventEntry(rkentry.EventEntryNoop),
		WithPort(8080),
		WithServerOptions(grpc.MaxRecvMsgSize(10)),
		WithUnaryInterceptors(rkgrpcmeta.UnaryServerInterceptor()),
		WithStreamInterceptors(rkgrpcmeta.StreamServerInterceptor()),
		WithGrpcRegF(func(server *grpc.Server) {}),
		WithCertEntry(certEntry),
		WithCommonServiceEntry(rkentry.RegisterCommonServiceEntry(&rkentry.BootCommonService{
			Enabled: true,
		})),
		WithEnableReflection(true),
		WithSwEntry(rkentry.RegisterSWEntry(&rkentry.BootSW{
			Enabled: true,
		})),
		WithProxyEntry(NewProxyEntry()),
		WithPromEntry(rkentry.RegisterPromEntry(&rkentry.BootProm{
			Enabled: true,
		})),
		WithStaticFileHandlerEntry(rkentry.RegisterStaticFileHandlerEntry(&rkentry.BootStaticFileHandler{
			Enabled: true,
		})),
		WithGwRegF(func(ctx context.Context, mux *gwruntime.ServeMux, s string, options []grpc.DialOption) error {
			return nil
		}),
		WithGrpcDialOptions(grpc.WithBlock()),
		WithGwMuxOptions(gwruntime.WithDisablePathLengthFallback()),
	)
	assert.True(t, entry.IsCommonServiceEnabled())
	assert.True(t, entry.IsProxyEnabled())
	assert.True(t, entry.IsSWEnabled())
	assert.True(t, entry.IsStaticFileHandlerEnabled())
	assert.True(t, entry.IsPromEnabled())
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
	certEntry := rkentry.RegisterCertEntry(&rkentry.BootCert{
		Cert: []*rkentry.BootCertE{
			{
				Name: "ut-cert",
			},
		},
	})
	certEntry[0].Bootstrap(context.TODO())
	entry = RegisterGrpcEntry(
		WithServerOptions(grpc.MaxRecvMsgSize(10)),
		WithCertEntry(certEntry[0]))
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
		WithSwEntry(rkentry.RegisterSWEntry(&rkentry.BootSW{
			Enabled: true,
		})),
		WithStaticFileHandlerEntry(rkentry.RegisterStaticFileHandlerEntry(&rkentry.BootStaticFileHandler{
			Enabled: true,
		})),
		WithPromEntry(rkentry.RegisterPromEntry(&rkentry.BootProm{
			Enabled: true,
		})),
		WithCommonServiceEntry(rkentry.RegisterCommonServiceEntry(&rkentry.BootCommonService{
			Enabled: true,
		})))
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

func TestGetGrpcEntry(t *testing.T) {
	RegisterGrpcEntry(WithName("ut"))

	assert.NotNil(t, GetGrpcEntry("ut"))
	assert.Nil(t, GetGrpcEntry("not-exist"))
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
