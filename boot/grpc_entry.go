// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpc an implementation of rkentry.Entry which could be used start restful server with grpc framework
package rkgrpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-entry/middleware"
	"github.com/rookie-ninja/rk-entry/middleware/auth"
	"github.com/rookie-ninja/rk-entry/middleware/cors"
	"github.com/rookie-ninja/rk-entry/middleware/csrf"
	"github.com/rookie-ninja/rk-entry/middleware/jwt"
	"github.com/rookie-ninja/rk-entry/middleware/log"
	"github.com/rookie-ninja/rk-entry/middleware/meta"
	"github.com/rookie-ninja/rk-entry/middleware/metrics"
	"github.com/rookie-ninja/rk-entry/middleware/panic"
	"github.com/rookie-ninja/rk-entry/middleware/ratelimit"
	"github.com/rookie-ninja/rk-entry/middleware/secure"
	"github.com/rookie-ninja/rk-entry/middleware/timeout"
	"github.com/rookie-ninja/rk-entry/middleware/tracing"
	apiutil "github.com/rookie-ninja/rk-grpc/boot/api/third_party/gen/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor/auth"
	"github.com/rookie-ninja/rk-grpc/interceptor/cors"
	"github.com/rookie-ninja/rk-grpc/interceptor/csrf"
	"github.com/rookie-ninja/rk-grpc/interceptor/jwt"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/meta"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"github.com/rookie-ninja/rk-grpc/interceptor/ratelimit"
	"github.com/rookie-ninja/rk-grpc/interceptor/secure"
	"github.com/rookie-ninja/rk-grpc/interceptor/timeout"
	"github.com/rookie-ninja/rk-grpc/interceptor/tracing/telemetry"
	"github.com/rookie-ninja/rk-query"
	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"math"
	"net"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// This must be declared in order to register registration function into rk context
// otherwise, rk-boot won't able to bootstrap grpc entry automatically from boot config file
func init() {
	rkentry.RegisterEntryRegFunc(RegisterGrpcEntriesWithConfig)
}

const (
	// GrpcEntryType default entry type
	GrpcEntryType = "gRPC"
	// GrpcEntryDescription default entry description
	GrpcEntryDescription = "Internal RK entry which helps to bootstrap with Grpc framework."
)

// BootConfig Boot config which is for grpc entry.
type BootConfig struct {
	Grpc []struct {
		Name               string                          `yaml:"name" json:"name"`
		Description        string                          `yaml:"description" json:"description"`
		Port               uint64                          `yaml:"port" json:"port"`
		Enabled            bool                            `yaml:"enabled" json:"enabled"`
		EnableReflection   bool                            `yaml:"enableReflection" json:"enableReflection"`
		NoRecvMsgSizeLimit bool                            `yaml:"noRecvMsgSizeLimit" json:"noRecvMsgSizeLimit"`
		CertEntry          string                          `yaml:"certEntry" json:"certEntry"`
		CommonService      rkentry.BootConfigCommonService `yaml:"commonService" json:"commonService"`
		Sw                 rkentry.BootConfigSw            `yaml:"sw" json:"sw"`
		Tv                 rkentry.BootConfigTv            `yaml:"tv" json:"tv"`
		Prom               rkentry.BootConfigProm          `yaml:"prom" json:"prom"`
		Static             rkentry.BootConfigStaticHandler `yaml:"static" json:"static"`
		Proxy              BootConfigProxy                 `yaml:"proxy" json:"proxy"`
		EnableRkGwOption   bool                            `yaml:"enableRkGwOption" json:"enableRkGwOption"`
		GwOption           *gwOption                       `yaml:"gwOption" json:"gwOption"`
		GwMappingFilePaths []string                        `yaml:"gwMappingFilePaths" json:"gwMappingFilePaths"`
		Interceptors       struct {
			LoggingZap       rkmidlog.BootConfig     `yaml:"loggingZap" json:"loggingZap"`
			MetricsProm      rkmidmetrics.BootConfig `yaml:"metricsProm" json:"metricsProm"`
			Auth             rkmidauth.BootConfig    `yaml:"auth" json:"auth"`
			Cors             rkmidcors.BootConfig    `yaml:"cors" json:"cors"`
			Secure           rkmidsec.BootConfig     `yaml:"secure" json:"secure"`
			Meta             rkmidmeta.BootConfig    `yaml:"meta" json:"meta"`
			Jwt              rkmidjwt.BootConfig     `yaml:"jwt" json:"jwt"`
			Csrf             rkmidcsrf.BootConfig    `yaml:"csrf" yaml:"csrf"`
			RateLimit        rkmidlimit.BootConfig   `yaml:"rateLimit" json:"rateLimit"`
			Timeout          rkmidtimeout.BootConfig `yaml:"timeout" json:"timeout"`
			TracingTelemetry rkmidtrace.BootConfig   `yaml:"tracingTelemetry" json:"tracingTelemetry"`
		} `yaml:"interceptors" json:"interceptors"`
		Logger struct {
			ZapLogger   string `yaml:"zapLogger" json:"zapLogger"`
			EventLogger string `yaml:"eventLogger" json:"eventLogger"`
		} `yaml:"logger" json:"logger"`
	} `yaml:"grpc" json:"grpc"`
}

// GrpcEntry implements rkentry.Entry interface.
type GrpcEntry struct {
	EntryName         string                    `json:"entryName" yaml:"entryName"`
	EntryType         string                    `json:"entryType" yaml:"entryType"`
	EntryDescription  string                    `json:"-" yaml:"-"`
	ZapLoggerEntry    *rkentry.ZapLoggerEntry   `json:"-" yaml:"-"`
	EventLoggerEntry  *rkentry.EventLoggerEntry `json:"-" yaml:"-"`
	Port              uint64                    `json:"port" yaml:"port"`
	TlsConfig         *tls.Config               `json:"-" yaml:"-"`
	TlsConfigInsecure *tls.Config               `json:"-" yaml:"-"`
	// GRPC related
	Server             *grpc.Server                   `json:"-" yaml:"-"`
	ServerOpts         []grpc.ServerOption            `json:"-" yaml:"-"`
	UnaryInterceptors  []grpc.UnaryServerInterceptor  `json:"-" yaml:"-"`
	StreamInterceptors []grpc.StreamServerInterceptor `json:"-" yaml:"-"`
	GrpcRegF           []GrpcRegFunc                  `json:"-" yaml:"-"`
	EnableReflection   bool                           `json:"enableReflection" yaml:"enableReflection"`
	// Gateway related
	HttpMux             *http.ServeMux             `json:"-" yaml:"-"`
	HttpServer          *http.Server               `json:"-" yaml:"-"`
	GwMux               *gwruntime.ServeMux        `json:"-" yaml:"-"`
	GwMuxOptions        []gwruntime.ServeMuxOption `json:"-" yaml:"-"`
	GwRegF              []GwRegFunc                `json:"-" yaml:"-"`
	GwMappingFilePaths  []string                   `json:"gwMappingFilePaths" yaml:"gwMappingFilePaths"`
	GwDialOptions       []grpc.DialOption          `json:"-" yaml:"-"`
	GwHttpToGrpcMapping map[string]*gwRule         `json:"gwMapping" yaml:"gwMapping"`
	gwCorsOptions       []rkmidcors.Option         `json:"-" yaml:"-"`
	gwSecureOptions     []rkmidsec.Option          `json:"-" yaml:"-"`
	gwCsrfOptions       []rkmidcsrf.Option         `json:"-" yaml:"-"`
	// Utility related
	SwEntry            *rkentry.SwEntry                `json:"-" yaml:"-"`
	TvEntry            *rkentry.TvEntry                `json:"-" yaml:"-"`
	ProxyEntry         *ProxyEntry                     `json:"-" yaml:"-"`
	PromEntry          *rkentry.PromEntry              `json:"-" yaml:"-"`
	StaticFileEntry    *rkentry.StaticFileHandlerEntry `json:"-" yaml:"-"`
	CommonServiceEntry *rkentry.CommonServiceEntry     `json:"-" yaml:"-"`
	CertEntry          *rkentry.CertEntry              `json:"-" yaml:"-"`
}

// RegisterGrpcEntriesWithConfig Register grpc entries with provided config file (Must YAML file).
//
// Currently, support two ways to provide config file path.
// 1: With function parameters
// 2: With command line flag "--rkboot" described in rkcommon.BootConfigPathFlagKey (Will override function parameter if exists)
// Command line flag has high priority which would override function parameter
//
// Error handling:
// Process will shutdown if any errors occur with rkcommon.ShutdownWithError function
//
// Override elements in config file:
// We learned from HELM source code which would override elements in YAML file with "--set" flag followed with comma
// separated key/value pairs.
//
// We are using "--rkset" described in rkcommon.BootConfigOverrideKey in order to distinguish with user flags
// Example of common usage: ./binary_file --rkset "key1=val1,key2=val2"
// Example of nested map:   ./binary_file --rkset "outer.inner.key=val"
// Example of slice:        ./binary_file --rkset "outer[0].key=val"
func RegisterGrpcEntriesWithConfig(configFilePath string) map[string]rkentry.Entry {
	res := make(map[string]rkentry.Entry)

	// 1: decode config map into boot config struct
	config := &BootConfig{}
	rkcommon.UnmarshalBootConfig(configFilePath, config)

	for i := range config.Grpc {
		element := config.Grpc[i]
		if !element.Enabled {
			continue
		}

		zapLoggerEntry := rkentry.GlobalAppCtx.GetZapLoggerEntry(element.Logger.ZapLogger)
		if zapLoggerEntry == nil {
			zapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
		}

		eventLoggerEntry := rkentry.GlobalAppCtx.GetEventLoggerEntry(element.Logger.EventLogger)
		if eventLoggerEntry == nil {
			eventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
		}

		// Register common service entry
		commonServiceEntry := rkentry.RegisterCommonServiceEntryWithConfig(&element.CommonService, element.Name,
			zapLoggerEntry, eventLoggerEntry)

		// Register swagger entry
		swEntry := rkentry.RegisterSwEntryWithConfig(&element.Sw, element.Name, element.Port,
			zapLoggerEntry, eventLoggerEntry, element.CommonService.Enabled)

		// Register TV entry
		tvEntry := rkentry.RegisterTvEntryWithConfig(&element.Tv, element.Name,
			zapLoggerEntry, eventLoggerEntry)

		// Register static file handler
		staticEntry := rkentry.RegisterStaticFileHandlerEntryWithConfig(&element.Static, element.Name,
			zapLoggerEntry, eventLoggerEntry)

		// Did we enabled proxy?
		var proxy *ProxyEntry
		if element.Proxy.Enabled {
			opts := make([]ruleOption, 0)
			for i := range element.Proxy.Rules {
				rule := element.Proxy.Rules[i]
				switch rule.Type {
				case HeaderBased:
					headers := make(map[string]string, 0)

					for i := range rule.HeaderPairs {
						tokens := strings.SplitN(rule.HeaderPairs[i], ":", 2)
						if len(tokens) != 2 {
							continue
						}
						headers[tokens[0]] = tokens[1]
					}

					opts = append(opts, WithHeaderPatterns(&HeaderPattern{
						Headers: headers,
						Dest:    rule.Dest,
					}))

				case PathBased:
					opts = append(opts, WithPathPatterns(&PathPattern{
						Paths: rule.Paths,
						Dest:  rule.Dest,
					}))
				case IpBased:
					opts = append(opts, WithIpPatterns(&IpPattern{
						Cidrs: rule.Ips,
						Dest:  rule.Dest,
					}))
				}
			}

			proxy = NewProxyEntry(
				WithNameProxy(element.Name),
				WithEventLoggerEntryProxy(eventLoggerEntry),
				WithZapLoggerEntryProxy(zapLoggerEntry),
				WithRuleProxy(NewRule(opts...)))
		}

		// Register prometheus entry
		promRegistry := prometheus.NewRegistry()
		promEntry := rkentry.RegisterPromEntryWithConfig(&element.Prom, element.Name, element.Port,
			zapLoggerEntry, eventLoggerEntry, promRegistry)

		var grpcDialOptions = make([]grpc.DialOption, 0)
		var gwMuxOpts = make([]gwruntime.ServeMuxOption, 0)
		if element.EnableRkGwOption {
			mOpt := mergeWithRkGwMarshalOption(element.GwOption)
			uOpt := mergeWithRkGwUnmarshalOption(element.GwOption)
			gwMuxOpts = append(gwMuxOpts, NewRkGwServerMuxOptions(mOpt, uOpt)...)
		} else {
			gwMuxOpts = append(gwMuxOpts, gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, &gwruntime.JSONPb{
				MarshalOptions:   *toMarshalOptions(element.GwOption),
				UnmarshalOptions: *toUnmarshalOptions(element.GwOption),
			}))
		}

		entry := RegisterGrpcEntry(
			WithName(element.Name),
			WithDescription(element.Description),
			WithZapLoggerEntry(zapLoggerEntry),
			WithEventLoggerEntry(eventLoggerEntry),
			WithPort(element.Port),
			WithGrpcDialOptions(grpcDialOptions...),
			WithSwEntry(swEntry),
			WithTvEntry(tvEntry),
			WithPromEntry(promEntry),
			WithProxyEntry(proxy),
			WithGwMuxOptions(gwMuxOpts...),
			WithCommonServiceEntry(commonServiceEntry),
			WithStaticFileHandlerEntry(staticEntry),
			WithEnableReflection(element.EnableReflection),
			WithGwMappingFilePaths(element.GwMappingFilePaths...),
			WithCertEntry(rkentry.GlobalAppCtx.GetCertEntry(element.CertEntry)))

		// Did we disabled message size for receiving?
		if element.NoRecvMsgSizeLimit {
			entry.ServerOpts = append(entry.ServerOpts, grpc.MaxRecvMsgSize(math.MaxInt64))
			entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(math.MaxInt64),
				grpc.MaxCallRecvMsgSize(math.MaxInt64)))
		}

		// logging middleware
		if element.Interceptors.LoggingZap.Enabled {
			entry.AddUnaryInterceptors(rkgrpclog.UnaryServerInterceptor(
				rkmidlog.ToOptions(&element.Interceptors.LoggingZap, element.Name, GrpcEntryType,
					zapLoggerEntry, eventLoggerEntry)...))
			entry.AddStreamInterceptors(rkgrpclog.StreamServerInterceptor(
				rkmidlog.ToOptions(&element.Interceptors.LoggingZap, element.Name, GrpcEntryType,
					zapLoggerEntry, eventLoggerEntry)...))
		}

		// did we enabled metrics interceptor?
		if element.Interceptors.MetricsProm.Enabled {
			entry.AddUnaryInterceptors(rkgrpcmetrics.UnaryServerInterceptor(
				rkmidmetrics.ToOptions(&element.Interceptors.MetricsProm, element.Name, GrpcEntryType,
					promRegistry, rkmidmetrics.LabelerTypeGrpc)...))
			entry.AddStreamInterceptors(rkgrpcmetrics.StreamServerInterceptor(
				rkmidmetrics.ToOptions(&element.Interceptors.MetricsProm, element.Name, GrpcEntryType,
					promRegistry, rkmidmetrics.LabelerTypeGrpc)...))
		}

		// trace middleware
		if element.Interceptors.TracingTelemetry.Enabled {
			entry.AddUnaryInterceptors(rkgrpctrace.UnaryServerInterceptor(
				rkmidtrace.ToOptions(&element.Interceptors.TracingTelemetry, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpctrace.StreamServerInterceptor(
				rkmidtrace.ToOptions(&element.Interceptors.TracingTelemetry, element.Name, GrpcEntryType)...))
		}

		// jwt middleware
		if element.Interceptors.Jwt.Enabled {
			entry.AddUnaryInterceptors(rkgrpcjwt.UnaryServerInterceptor(
				rkmidjwt.ToOptions(&element.Interceptors.Jwt, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpcjwt.StreamServerInterceptor(
				rkmidjwt.ToOptions(&element.Interceptors.Jwt, element.Name, GrpcEntryType)...))
		}

		// secure middleware
		if element.Interceptors.Secure.Enabled {
			entry.AddGwSecureOptions(rkmidsec.ToOptions(
				&element.Interceptors.Secure, element.Name, GrpcEntryType)...)
		}

		// csrf middleware
		if element.Interceptors.Csrf.Enabled {
			entry.AddGwCsrfOptions(rkmidcsrf.ToOptions(
				&element.Interceptors.Csrf, element.Name, GrpcEntryType)...)
		}

		// cors middleware
		if element.Interceptors.Cors.Enabled {
			entry.AddGwCorsOptions(rkmidcors.ToOptions(
				&element.Interceptors.Cors, element.Name, GrpcEntryType)...)
		}

		// meta middleware
		if element.Interceptors.Meta.Enabled {
			entry.AddUnaryInterceptors(rkgrpcmeta.UnaryServerInterceptor(
				rkmidmeta.ToOptions(&element.Interceptors.Meta, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpcmeta.StreamServerInterceptor(
				rkmidmeta.ToOptions(&element.Interceptors.Meta, element.Name, GrpcEntryType)...))
		}

		// auth middleware
		if element.Interceptors.Auth.Enabled {
			entry.AddUnaryInterceptors(rkgrpcauth.UnaryServerInterceptor(
				rkmidauth.ToOptions(&element.Interceptors.Auth, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpcauth.StreamServerInterceptor(
				rkmidauth.ToOptions(&element.Interceptors.Auth, element.Name, GrpcEntryType)...))
		}

		// timeout middleware
		if element.Interceptors.Timeout.Enabled {
			entry.AddUnaryInterceptors(rkgrpctimeout.UnaryServerInterceptor(
				rkmidtimeout.ToOptions(&element.Interceptors.Timeout, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpctimeout.StreamServerInterceptor(
				rkmidtimeout.ToOptions(&element.Interceptors.Timeout, element.Name, GrpcEntryType)...))
		}

		// ratelimit middleware
		if element.Interceptors.RateLimit.Enabled {
			entry.AddUnaryInterceptors(rkgrpclimit.UnaryServerInterceptor(
				rkmidlimit.ToOptions(&element.Interceptors.RateLimit, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpclimit.StreamServerInterceptor(
				rkmidlimit.ToOptions(&element.Interceptors.RateLimit, element.Name, GrpcEntryType)...))
		}

		res[element.Name] = entry
	}
	return res
}

// RegisterGrpcEntry Register GrpcEntry with options.
func RegisterGrpcEntry(opts ...GrpcEntryOption) *GrpcEntry {
	entry := &GrpcEntry{
		EntryType:        GrpcEntryType,
		EntryDescription: GrpcEntryDescription,
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		Port:             8080,
		// gRPC related
		ServerOpts:         make([]grpc.ServerOption, 0),
		UnaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		StreamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		GrpcRegF:           make([]GrpcRegFunc, 0),
		EnableReflection:   true,
		// grpc-gateway related
		GwMuxOptions:        make([]gwruntime.ServeMuxOption, 0),
		GwRegF:              make([]GwRegFunc, 0),
		GwMappingFilePaths:  make([]string, 0),
		GwHttpToGrpcMapping: make(map[string]*gwRule),
		GwDialOptions:       make([]grpc.DialOption, 0),
		HttpMux:             http.NewServeMux(),
		gwCorsOptions:       make([]rkmidcors.Option, 0),
		gwCsrfOptions:       make([]rkmidcsrf.Option, 0),
		gwSecureOptions:     make([]rkmidsec.Option, 0),
	}

	for i := range opts {
		opts[i](entry)
	}

	entry.UnaryInterceptors = append(entry.UnaryInterceptors, rkgrpcpanic.UnaryServerInterceptor(
		rkmidpanic.WithEntryNameAndType(entry.EntryName, entry.EntryType)))
	entry.StreamInterceptors = append(entry.StreamInterceptors, rkgrpcpanic.StreamServerInterceptor(
		rkmidpanic.WithEntryNameAndType(entry.EntryName, entry.EntryType)))

	if entry.ZapLoggerEntry == nil {
		entry.ZapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
	}

	if entry.EventLoggerEntry == nil {
		entry.EventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = "GrpcServer-" + strconv.FormatUint(entry.Port, 10)
	}

	// Init TLS config
	if entry.IsTlsEnabled() {
		var cert tls.Certificate
		var err error
		if cert, err = tls.X509KeyPair(entry.CertEntry.Store.ServerCert, entry.CertEntry.Store.ServerKey); err != nil {
			entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing TLS.", zap.String("cert", entry.CertEntry.String()))
		} else {
			entry.TlsConfig = &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cert},
			}
			entry.TlsConfigInsecure = &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cert},
			}
		}
	}

	// add entry name and entry type into loki syncer if enabled
	entry.ZapLoggerEntry.AddEntryLabelToLokiSyncer(entry)
	entry.EventLoggerEntry.AddEntryLabelToLokiSyncer(entry)

	rkentry.GlobalAppCtx.AddEntry(entry)

	return entry
}

// ************* Entry function *************

// GetName Get entry name.
func (entry *GrpcEntry) GetName() string {
	return entry.EntryName
}

// GetType Get entry type.
func (entry *GrpcEntry) GetType() string {
	return entry.EntryType
}

// String Stringfy entry.
func (entry *GrpcEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// GetDescription Get description of entry.
func (entry *GrpcEntry) GetDescription() string {
	return entry.EntryDescription
}

// Bootstrap GrpcEntry.
func (entry *GrpcEntry) Bootstrap(ctx context.Context) {
	event, logger := entry.logBasicInfo("Bootstrap")

	// 1: Parse gateway mapping file paths, this will record http to grpc path map into a map
	// which will be used for /apis call in CommonServiceEntry
	entry.parseGwMapping()

	// 2: Create grpc server
	// 2.1: Make unary and stream interceptors into server opts
	// Important! Do not add tls as options since we already enable tls in listener
	entry.ServerOpts = append(entry.ServerOpts,
		grpc.ChainUnaryInterceptor(entry.UnaryInterceptors...),
		grpc.ChainStreamInterceptor(entry.StreamInterceptors...))

	// 3: Add proxy entry
	if entry.IsProxyEnabled() {
		entry.ServerOpts = append(entry.ServerOpts,
			grpc.ForceServerCodec(Codec()),
			grpc.UnknownServiceHandler(TransparentHandler(entry.ProxyEntry.r.GetDirector())),
		)
		entry.ProxyEntry.Bootstrap(ctx)
	}

	// 4: Create grpc server
	entry.Server = grpc.NewServer(entry.ServerOpts...)

	// 5: Register grpc function into server
	for _, regFunc := range entry.GrpcRegF {
		regFunc(entry.Server)
	}

	// 6: Enable grpc reflection
	if entry.EnableReflection {
		reflection.Register(entry.Server)
	}

	// 7: Create http server based on grpc gateway
	// 7.1: Create gateway mux
	entry.GwMux = gwruntime.NewServeMux(entry.GwMuxOptions...)

	// 8: Inject insecure option into dial option since grpc call is delegated from gateway which is inner code call
	// and which is safe!
	if entry.TlsConfig != nil {
		entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithTransportCredentials(credentials.NewTLS(entry.TlsConfigInsecure)))
	} else {
		entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithInsecure())
	}

	// 9: Register grpc gateway function into GwMux
	for i := range entry.GwRegF {
		err := entry.GwRegF[i](context.Background(), entry.GwMux, "0.0.0.0:"+strconv.FormatUint(entry.Port, 10), entry.GwDialOptions)
		if err != nil {
			entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
			rkcommon.ShutdownWithError(err)
		}
	}

	// 10: Make http mux listen on path of / and configure TV, swagger, prometheus path
	entry.HttpMux.Handle("/", entry.GwMux)

	// 11: swagger
	if entry.IsSwEnabled() {
		entry.HttpMux.HandleFunc(entry.SwEntry.Path, entry.SwEntry.ConfigFileHandler())
		entry.HttpMux.HandleFunc(entry.SwEntry.AssetsFilePath, entry.SwEntry.AssetsFileHandler())

		entry.SwEntry.Bootstrap(ctx)
	}

	// 12: tv
	if entry.IsTvEnabled() {
		entry.HttpMux.HandleFunc(entry.TvEntry.BasePath, entry.TV)
		entry.HttpMux.HandleFunc(entry.TvEntry.AssetsFilePath, entry.TvEntry.AssetsFileHandler())

		entry.TvEntry.Bootstrap(ctx)
	}

	// 13: static file handler
	if entry.IsStaticFileHandlerEnabled() {
		entry.HttpMux.HandleFunc(entry.StaticFileEntry.Path, entry.StaticFileEntry.GetFileHandler())

		entry.StaticFileEntry.Bootstrap(ctx)
	}

	// 14: prometheus
	if entry.IsPromEnabled() {
		// Register prom path into Router.
		entry.HttpMux.Handle(entry.PromEntry.Path, promhttp.HandlerFor(entry.PromEntry.Gatherer, promhttp.HandlerOpts{}))

		entry.PromEntry.Bootstrap(ctx)
	}

	// 15: common service
	if entry.IsCommonServiceEnabled() {
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.HealthyPath, entry.CommonServiceEntry.Healthy)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.GcPath, entry.CommonServiceEntry.Gc)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.InfoPath, entry.CommonServiceEntry.Info)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.ConfigsPath, entry.CommonServiceEntry.Configs)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.SysPath, entry.CommonServiceEntry.Sys)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.EntriesPath, entry.CommonServiceEntry.Entries)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.CertsPath, entry.CommonServiceEntry.Certs)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.LogsPath, entry.CommonServiceEntry.Logs)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.DepsPath, entry.CommonServiceEntry.Deps)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.LicensePath, entry.CommonServiceEntry.License)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.ReadmePath, entry.CommonServiceEntry.Readme)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.GitPath, entry.CommonServiceEntry.Git)

		// swagger doc already generated at rkentry.CommonService
		// follow bellow actions
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.ApisPath, entry.Apis)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.ReqPath, entry.Req)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.GwErrorMappingPath, entry.GwErrorMapping)

		// Bootstrap common service entry.
		entry.CommonServiceEntry.Bootstrap(ctx)
	}

	// 16: Create http server
	var httpHandler http.Handler
	httpHandler = entry.HttpMux

	// 17: If CORS enabled, then add interceptor for grpc-gateway
	if len(entry.gwCorsOptions) > 0 {
		httpHandler = rkgrpccors.Interceptor(httpHandler, entry.gwCorsOptions...)
	}

	// 18: If Secure enabled, then add interceptor for grpc-gateway
	if len(entry.gwSecureOptions) > 0 {
		httpHandler = rkgrpcsec.Interceptor(httpHandler, entry.gwSecureOptions...)
	}

	// 19: If CSRF enabled, then add interceptor for grpc-gateway
	if len(entry.gwCsrfOptions) > 0 {
		httpHandler = rkgrpccsrf.Interceptor(httpHandler, entry.gwCsrfOptions...)
	}

	entry.HttpServer = &http.Server{
		Addr:    "0.0.0.0:" + strconv.FormatUint(entry.Port, 10),
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}),
	}

	// 20: Start http server
	entry.EventLoggerEntry.GetEventHelper().Finish(event)
	go func(*GrpcEntry) {
		// Create inner listener
		conn, err := net.Listen("tcp4", ":"+strconv.FormatUint(entry.Port, 10))
		if err != nil {
			entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
			rkcommon.ShutdownWithError(err)
		}

		// We will use cmux to make grpc and grpc gateway on the same port.
		// With cmux, we can init one listener but routes connection based on some rules.
		if !entry.IsTlsEnabled() {
			// 1: Create a TCP listener with cmux
			tcpL := cmux.New(conn)

			// 2: If header value of content-type is application/grpc, then it is a grpc request.
			// Assign a wrapped listener to grpc connection with cmux
			grpcL := tcpL.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))

			// 3: Not a grpc connection, we will wrap a http listener.
			httpL := tcpL.Match(cmux.HTTP1Fast())

			// 4: Start both of grpc and http server
			go entry.startGrpcServer(grpcL, logger)
			go entry.startHttpServer(httpL, logger)

			// 5: Start listener
			if err := tcpL.Serve(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				if err != cmux.ErrListenerClosed {
					event.AddErr(err)
					logger.Error("Error occurs while serving TCP listener.", zap.Error(err))
					rkcommon.ShutdownWithError(err)
				}
			}
		} else {
			// In this case, we will enable tls
			// 1: Create a tls listener with tls config
			tlsL := cmux.New(tls.NewListener(conn, entry.TlsConfig))

			// 2: If header value of content-type is application/grpc, then it is a grpc request.
			// Assign a wrapped listener to grpc connection with cmux
			grpcL := tlsL.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))

			// 3: Not a grpc connection, we will wrap a http listener.
			httpL := tlsL.Match(cmux.HTTP1Fast())

			// 4: Start both of grpc and http server
			go entry.startGrpcServer(grpcL, logger)
			go entry.startHttpServer(httpL, logger)

			// 5: Start listener
			if err := tlsL.Serve(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				if err != cmux.ErrListenerClosed {
					event.AddErr(err)
					logger.Error("Error occurs while serving TLS listener.", zap.Error(err))
					rkcommon.ShutdownWithError(err)
				}
			}
		}
	}(entry)
}

func (entry *GrpcEntry) startGrpcServer(lis net.Listener, logger *zap.Logger) {
	if err := entry.Server.Serve(lis); err != nil && !strings.Contains(err.Error(), "mux: server closed") {
		logger.Error("Error occurs while serving grpc-server.", zap.Error(err))
		rkcommon.ShutdownWithError(err)
	}
}

func (entry *GrpcEntry) startHttpServer(lis net.Listener, logger *zap.Logger) {
	if err := entry.HttpServer.Serve(lis); err != nil && !strings.Contains(err.Error(), "http: Server closed") {
		logger.Error("Error occurs while serving gateway-server.", zap.Error(err))
		rkcommon.ShutdownWithError(err)
	}
}

// Interrupt GrpcEntry.
func (entry *GrpcEntry) Interrupt(ctx context.Context) {
	event, logger := entry.logBasicInfo("Interrupt")

	// 3: Interrupt CommonServiceEntry, SwEntry, TvEntry, PromEntry
	if entry.IsCommonServiceEnabled() {
		entry.CommonServiceEntry.Interrupt(ctx)
	}

	if entry.IsSwEnabled() {
		entry.SwEntry.Interrupt(ctx)
	}

	if entry.IsStaticFileHandlerEnabled() {
		entry.StaticFileEntry.Interrupt(ctx)
	}

	if entry.IsTvEnabled() {
		entry.TvEntry.Interrupt(ctx)
	}

	if entry.IsPromEnabled() {
		entry.PromEntry.Interrupt(ctx)
	}

	if entry.HttpServer != nil {
		if err := entry.HttpServer.Shutdown(context.Background()); err != nil {
			event.AddErr(err)
			logger.Warn("Error occurs while stopping http server")
		}
	}

	if entry.Server != nil {
		entry.Server.GracefulStop()
	}

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)
}

// ************* public function *************

// AddServerOptions Add grpc server options.
func (entry *GrpcEntry) AddServerOptions(opts ...grpc.ServerOption) {
	entry.ServerOpts = append(entry.ServerOpts, opts...)
}

// AddUnaryInterceptors Add unary interceptor.
func (entry *GrpcEntry) AddUnaryInterceptors(inter ...grpc.UnaryServerInterceptor) {
	entry.UnaryInterceptors = append(entry.UnaryInterceptors, inter...)
}

// AddStreamInterceptors Add stream interceptor.
func (entry *GrpcEntry) AddStreamInterceptors(inter ...grpc.StreamServerInterceptor) {
	entry.StreamInterceptors = append(entry.StreamInterceptors, inter...)
}

// AddGwCorsOptions Enable CORS at gateway side with options.
func (entry *GrpcEntry) AddGwCorsOptions(opts ...rkmidcors.Option) {
	entry.gwCorsOptions = append(entry.gwCorsOptions, opts...)
}

// AddGwCsrfOptions Enable CORS at gateway side with options.
func (entry *GrpcEntry) AddGwCsrfOptions(opts ...rkmidcsrf.Option) {
	entry.gwCsrfOptions = append(entry.gwCsrfOptions, opts...)
}

// AddGwSecureOptions Enable secure at gateway side with options.
func (entry *GrpcEntry) AddGwSecureOptions(opts ...rkmidsec.Option) {
	entry.gwSecureOptions = append(entry.gwSecureOptions, opts...)
}

// AddRegFuncGrpc Add grpc registration func.
func (entry *GrpcEntry) AddRegFuncGrpc(f ...GrpcRegFunc) {
	entry.GrpcRegF = append(entry.GrpcRegF, f...)
}

// AddRegFuncGw Add gateway registration func.
func (entry *GrpcEntry) AddRegFuncGw(f ...GwRegFunc) {
	entry.GwRegF = append(entry.GwRegF, f...)
}

// AddGwDialOptions Add grpc dial options called from grpc gateway
func (entry *GrpcEntry) AddGwDialOptions(opts ...grpc.DialOption) {
	entry.GwDialOptions = append(entry.GwDialOptions, opts...)
}

// MarshalJSON Marshal entry.
func (entry *GrpcEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":          entry.EntryName,
		"entryType":          entry.EntryType,
		"entryDescription":   entry.EntryDescription,
		"eventLoggerEntry":   entry.EventLoggerEntry.GetName(),
		"zapLoggerEntry":     entry.ZapLoggerEntry.GetName(),
		"port":               entry.Port,
		"commonServiceEntry": entry.CommonServiceEntry,
		"swEntry":            entry.SwEntry,
		"tvEntry":            entry.TvEntry,
		"promEntry":          entry.PromEntry,
		"reflection":         entry.EnableReflection,
	}

	if entry.CertEntry != nil {
		m["certEntry"] = entry.CertEntry.GetName()
	}

	// Interceptors
	interceptorsStr := make([]string, 0)
	m["interceptors"] = &interceptorsStr

	for i := range entry.UnaryInterceptors {
		element := entry.UnaryInterceptors[i]
		interceptorsStr = append(interceptorsStr,
			path.Base(runtime.FuncForPC(reflect.ValueOf(element).Pointer()).Name()))
	}

	for i := range entry.StreamInterceptors {
		element := entry.StreamInterceptors[i]
		interceptorsStr = append(interceptorsStr,
			path.Base(runtime.FuncForPC(reflect.ValueOf(element).Pointer()).Name()))
	}

	serverOptsStr := make([]string, 0)
	m["serverOpts"] = &serverOptsStr

	for i := range entry.ServerOpts {
		element := entry.ServerOpts[i]
		serverOptsStr = append(serverOptsStr,
			runtime.FuncForPC(reflect.ValueOf(element).Pointer()).Name())
	}

	grpcRegFStr := make([]string, 0)
	m["grpcRegF"] = &grpcRegFStr
	for i := range entry.GrpcRegF {
		element := entry.GrpcRegF[i]
		grpcRegFStr = append(grpcRegFStr,
			runtime.FuncForPC(reflect.ValueOf(element).Pointer()).Name())
	}

	gwRegFStr := make([]string, 0)
	m["gwRegF"] = &gwRegFStr
	for i := range entry.GwRegF {
		element := entry.GwRegF[i]
		gwRegFStr = append(gwRegFStr,
			runtime.FuncForPC(reflect.ValueOf(element).Pointer()).Name())
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Not supported.
func (entry *GrpcEntry) UnmarshalJSON([]byte) error {
	return nil
}

// IsTlsEnabled Is TLS enabled?
func (entry *GrpcEntry) IsTlsEnabled() bool {
	return entry.CertEntry != nil && entry.CertEntry.Store != nil
}

// IsCommonServiceEnabled Is common service enabled?
func (entry *GrpcEntry) IsCommonServiceEnabled() bool {
	return entry.CommonServiceEntry != nil
}

// IsProxyEnabled Is proxy enabled?
func (entry *GrpcEntry) IsProxyEnabled() bool {
	return entry.ProxyEntry != nil
}

// IsSwEnabled Is swagger enabled?
func (entry *GrpcEntry) IsSwEnabled() bool {
	return entry.SwEntry != nil
}

// IsStaticFileHandlerEnabled Is static file handler entry enabled?
func (entry *GrpcEntry) IsStaticFileHandlerEnabled() bool {
	return entry.StaticFileEntry != nil
}

// IsTvEnabled Is tv enabled?
func (entry *GrpcEntry) IsTvEnabled() bool {
	return entry.TvEntry != nil
}

// IsPromEnabled Is prometheus client enabled?
func (entry *GrpcEntry) IsPromEnabled() bool {
	return entry.PromEntry != nil
}

// Add basic fields into event.
func (entry *GrpcEntry) logBasicInfo(operation string) (rkquery.Event, *zap.Logger) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		operation,
		rkquery.WithEntryName(entry.GetName()),
		rkquery.WithEntryType(entry.GetType()))
	logger := entry.ZapLoggerEntry.GetLogger().With(
		zap.String("eventId", event.GetEventId()),
		zap.String("entryName", entry.EntryName))

	// add general info
	event.AddPayloads(
		zap.Uint64("grpcPort", entry.Port),
		zap.Uint64("gwPort", entry.Port))

	// add SwEntry info
	if entry.IsSwEnabled() {
		event.AddPayloads(
			zap.Bool("swEnabled", true),
			zap.String("swPath", entry.SwEntry.Path))
	}

	// add CommonServiceEntry info
	if entry.IsCommonServiceEnabled() {
		event.AddPayloads(
			zap.Bool("commonServiceEnabled", true),
			zap.String("commonServicePathPrefix", "/rk/v1/"))
	}

	// add TvEntry info
	if entry.IsTvEnabled() {
		event.AddPayloads(
			zap.Bool("tvEnabled", true),
			zap.String("tvPath", "/rk/v1/tv/"))
	}

	// add PromEntry info
	if entry.IsPromEnabled() {
		event.AddPayloads(
			zap.Bool("promEnabled", true),
			zap.Uint64("promPort", entry.PromEntry.Port),
			zap.String("promPath", entry.PromEntry.Path))
	}

	// add StaticFileHandlerEntry info
	if entry.IsStaticFileHandlerEnabled() {
		event.AddPayloads(
			zap.Bool("staticFileHandlerEnabled", true),
			zap.String("staticFileHandlerPath", entry.StaticFileEntry.Path))
	}

	// add tls info
	if entry.IsTlsEnabled() {
		event.AddPayloads(
			zap.Bool("tlsEnabled", true))
	}

	// add proxy info
	if entry.IsProxyEnabled() {
		event.AddPayloads(
			zap.Bool("grpcProxyEnabled", true))
	}

	logger.Info(fmt.Sprintf("%s grpcEntry", operation))

	return event, logger
}

// Parse gw mapping file
func (entry *GrpcEntry) parseGwMapping() {
	// Parse user services.
	for i := range entry.GwMappingFilePaths {
		filePath := entry.GwMappingFilePaths[i]

		// case 1: read file
		bytes := rkcommon.TryReadFile(filePath)
		if len(bytes) < 1 {
			continue
		}

		// case 2: convert json to yaml
		jsonContents, err := yaml.YAMLToJSON(bytes)
		if err != nil {
			entry.ZapLoggerEntry.GetLogger().Warn("Failed to convert grpc api config.", zap.Error(err))
			continue
		}

		// case 3: unmarshal
		unmarshaler := protojson.UnmarshalOptions{
			DiscardUnknown: true,
		}
		mapping := &apiutil.GrpcAPIService{}
		if err := unmarshaler.Unmarshal(jsonContents, mapping); err != nil {
			entry.ZapLoggerEntry.GetLogger().Warn("Failed to parse grpc api config.", zap.Error(err))
			continue
		}

		// case 4: iterate rules
		rules := mapping.GetHttp().GetRules()
		for i := range rules {
			element := rules[i]
			rule := &gwRule{}
			entry.GwHttpToGrpcMapping[element.GetSelector()] = rule
			switch element.GetPattern().(type) {
			case *annotations.HttpRule_Get:
				rule.Pattern = strings.TrimSuffix(element.GetGet(), "/")
				rule.Method = "GET"
			case *annotations.HttpRule_Put:
				rule.Pattern = strings.TrimSuffix(element.GetPut(), "/")
				rule.Method = "PUT"
			case *annotations.HttpRule_Post:
				rule.Pattern = strings.TrimSuffix(element.GetPost(), "/")
				rule.Method = "POST"
			case *annotations.HttpRule_Delete:
				rule.Pattern = strings.TrimSuffix(element.GetDelete(), "/")
				rule.Method = "DELETE"
			case *annotations.HttpRule_Patch:
				rule.Pattern = strings.TrimSuffix(element.GetPatch(), "/")
				rule.Method = "PATCH"
			}
		}
	}
}

// GetGrpcEntry Get GrpcEntry from rkentry.GlobalAppCtx.
func GetGrpcEntry(name string) *GrpcEntry {
	if raw := rkentry.GlobalAppCtx.GetEntry(name); raw != nil {
		if res, ok := raw.(*GrpcEntry); ok {
			return res
		}
	}

	return nil
}

// ************** Common service extension **************

// Apis Stub
func (entry *GrpcEntry) Apis(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.WriteHeader(http.StatusOK)
	bytes, _ := json.MarshalIndent(entry.doApis(req), "", "  ")
	w.Write(bytes)
}

// Req Stub
func (entry *GrpcEntry) Req(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)

	mix := false
	if len(req.URL.Query().Get("fromTv")) > 0 {
		mix = true
	}

	bytes, _ := json.MarshalIndent(entry.doReq(mix), "", "  ")
	w.Write(bytes)
}

// GwErrorMapping Get error mapping file contents.
func (entry *GrpcEntry) GwErrorMapping(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	bytes, _ := json.MarshalIndent(entry.doGwErrorMapping(), "", "  ")
	w.Write(bytes)
}

// TV Http handler of /rk/v1/tv/*.
func (entry *GrpcEntry) TV(w http.ResponseWriter, req *http.Request) {
	logger := entry.ZapLoggerEntry.GetLogger()

	item := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/rk/v1/tv"), "/")

	w.Header().Set("charset", "utf-8")
	w.Header().Set("content-type", "text/html")

	switch item {
	case "/apis":
		buf := entry.TvEntry.ExecuteTemplate("apis", entry.doApis(req), logger)
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	case "/gwErrorMapping":
		buf := entry.TvEntry.ExecuteTemplate("gw-error-mapping", entry.doGwErrorMapping(), logger)
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	default:
		buf := entry.TvEntry.Action(item, logger)
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

func (entry *GrpcEntry) doApis(req *http.Request) *rkentry.ApisResponse {
	res := &rkentry.ApisResponse{
		Entries: make([]*rkentry.ApisResponseElement, 0),
	}

	for serviceName, serviceInfo := range entry.Server.GetServiceInfo() {
		for i := range serviceInfo.Methods {
			method := serviceInfo.Methods[i]

			entry := &rkentry.ApisResponseElement{
				EntryName: entry.GetName(),
				Method:    serviceName,
				Path:      method.Name,
				Gw:        entry.getGwMapping(serviceName + "." + method.Name),
				Port:      entry.Port,
				SwUrl:     entry.getSwUrl(req),
			}

			res.Entries = append(res.Entries, entry)
		}
	}

	return res
}

// Helper function for Req call
func (entry *GrpcEntry) doReq(mixGrpcAndRestApi bool) *rkentry.ReqResponse {
	metricsSet := rkmidmetrics.GetServerMetricsSet(entry.GetName())
	// case 1: nil metrics set
	if metricsSet == nil {
		return &rkentry.ReqResponse{
			Metrics: make([]*rkentry.ReqMetricsRK, 0),
		}
	}

	// case 2: nil vector
	vector := metricsSet.GetSummary(rkmidmetrics.MetricsNameElapsedNano)
	if vector == nil {
		return &rkentry.ReqResponse{
			Metrics: make([]*rkentry.ReqMetricsRK, 0),
		}
	}

	reqMetrics := rkentry.NewPromMetricsInfo(vector)

	// Fill missed metrics
	type innerGrpcInfo struct {
		grpcService string
		grpcMethod  string
	}

	apis := make([]*innerGrpcInfo, 0)

	infos := entry.Server.GetServiceInfo()
	for serviceName, serviceInfo := range infos {
		for j := range serviceInfo.Methods {
			apis = append(apis, &innerGrpcInfo{
				grpcService: serviceName,
				grpcMethod:  serviceInfo.Methods[j].Name,
			})
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

	// convert restful api to grpc
	if mixGrpcAndRestApi {
		for i := range reqMetrics {
			reqMetrics[i].RestPath = reqMetrics[i].GrpcMethod
			reqMetrics[i].RestMethod = reqMetrics[i].GrpcService
		}
	}

	return &rkentry.ReqResponse{
		Metrics: reqMetrics,
	}
}

// Helper function /gwErrorMapping
func (entry *GrpcEntry) doGwErrorMapping() *rkentry.GwErrorMappingResponse {
	res := &rkentry.GwErrorMappingResponse{
		Mapping: make(map[int32]*rkentry.GwErrorMappingResponseElement),
	}

	// list grpc errors
	for k, v := range code.Code_name {
		element := &rkentry.GwErrorMappingResponseElement{
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

// Compose gateway related elements based on GwEntry and SwEntry.
func (entry *GrpcEntry) getGwMapping(grpcMethod string) string {
	var value *gwRule
	var ok bool
	if value, ok = entry.GwHttpToGrpcMapping[grpcMethod]; !ok {
		return ""
	}

	return value.Method + " " + value.Pattern
}

// Compose swagger URL based on SwEntry.
func (entry *GrpcEntry) getSwUrl(req *http.Request) string {
	if entry.IsSwEnabled() {
		scheme := "http"
		if entry.IsTlsEnabled() {
			scheme = "https"
		}

		remoteIp, _ := rkmid.GetRemoteAddressSet(req)

		return fmt.Sprintf("%s://%s:%d%s",
			scheme,
			remoteIp,
			entry.SwEntry.Port,
			entry.SwEntry.Path)
	}

	return ""
}

// *********** Options ***********

// internal usage
type gwRule struct {
	Method  string `json:"method" yaml:"method"`
	Pattern string `json:"pattern" yaml:"pattern"`
}

// GwRegFunc Registration function grpc gateway.
type GwRegFunc func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error

// GrpcRegFunc Grpc registration func.
type GrpcRegFunc func(server *grpc.Server)

// GrpcEntryOption GrpcEntry option.
type GrpcEntryOption func(*GrpcEntry)

// WithName Provide name.
func WithName(name string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EntryName = name
	}
}

// WithDescription Provide description.
func WithDescription(description string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EntryDescription = description
	}
}

// WithZapLoggerEntry Provide rkentry.ZapLoggerEntry
func WithZapLoggerEntry(logger *rkentry.ZapLoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ZapLoggerEntry = logger
	}
}

// WithEventLoggerEntry Provide rkentry.EventLoggerEntry
func WithEventLoggerEntry(logger *rkentry.EventLoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EventLoggerEntry = logger
	}
}

// WithPort Provide port.
func WithPort(port uint64) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.Port = port
	}
}

// WithServerOptions Provide grpc.ServerOption.
func WithServerOptions(opts ...grpc.ServerOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ServerOpts = append(entry.ServerOpts, opts...)
	}
}

// WithUnaryInterceptors Provide grpc.UnaryServerInterceptor.
func WithUnaryInterceptors(opts ...grpc.UnaryServerInterceptor) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.UnaryInterceptors = append(entry.UnaryInterceptors, opts...)
	}
}

// WithStreamInterceptors Provide grpc.StreamServerInterceptor.
func WithStreamInterceptors(opts ...grpc.StreamServerInterceptor) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.StreamInterceptors = append(entry.StreamInterceptors, opts...)
	}
}

// WithGrpcRegF Provide GrpcRegFunc.
func WithGrpcRegF(f ...GrpcRegFunc) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GrpcRegF = append(entry.GrpcRegF, f...)
	}
}

// WithCertEntry Provide rkentry.CertEntry.
func WithCertEntry(certEntry *rkentry.CertEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.CertEntry = certEntry
	}
}

// WithCommonServiceEntry Provide CommonServiceEntry.
func WithCommonServiceEntry(commonService *rkentry.CommonServiceEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.CommonServiceEntry = commonService
	}
}

// WithEnableReflection Provide EnableReflection.
func WithEnableReflection(enabled bool) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EnableReflection = enabled
	}
}

// WithSwEntry Provide SwEntry.
func WithSwEntry(sw *rkentry.SwEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.SwEntry = sw
	}
}

// WithTvEntry Provide TvEntry.
func WithTvEntry(tv *rkentry.TvEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.TvEntry = tv
	}
}

// WithProxyEntry Provide ProxyEntry.
func WithProxyEntry(proxy *ProxyEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ProxyEntry = proxy
	}
}

// WithPromEntry Provide PromEntry.
func WithPromEntry(prom *rkentry.PromEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.PromEntry = prom
	}
}

// WithStaticFileHandlerEntry provide StaticFileHandlerEntry.
func WithStaticFileHandlerEntry(staticEntry *rkentry.StaticFileHandlerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.StaticFileEntry = staticEntry
	}
}

// WithGwRegF Provide registration function.
func WithGwRegF(f ...GwRegFunc) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwRegF = append(entry.GwRegF, f...)
	}
}

// WithGrpcDialOptions Provide grpc dial options.
func WithGrpcDialOptions(opts ...grpc.DialOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwDialOptions = append(entry.GwDialOptions, opts...)
	}
}

// WithGwMuxOptions Provide gateway server mux options.
func WithGwMuxOptions(opts ...gwruntime.ServeMuxOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwMuxOptions = append(entry.GwMuxOptions, opts...)
	}
}

// WithGwMappingFilePaths Provide gateway mapping configuration file paths.
func WithGwMappingFilePaths(paths ...string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwMappingFilePaths = append(entry.GwMappingFilePaths, paths...)
	}
}
