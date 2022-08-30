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
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rookie-ninja/rk-entry/v2/entry"
	rkerror "github.com/rookie-ninja/rk-entry/v2/error"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/auth"
	"github.com/rookie-ninja/rk-entry/v2/middleware/cors"
	"github.com/rookie-ninja/rk-entry/v2/middleware/csrf"
	"github.com/rookie-ninja/rk-entry/v2/middleware/jwt"
	"github.com/rookie-ninja/rk-entry/v2/middleware/log"
	"github.com/rookie-ninja/rk-entry/v2/middleware/meta"
	"github.com/rookie-ninja/rk-entry/v2/middleware/panic"
	"github.com/rookie-ninja/rk-entry/v2/middleware/prom"
	"github.com/rookie-ninja/rk-entry/v2/middleware/ratelimit"
	"github.com/rookie-ninja/rk-entry/v2/middleware/secure"
	"github.com/rookie-ninja/rk-entry/v2/middleware/timeout"
	"github.com/rookie-ninja/rk-entry/v2/middleware/tracing"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/auth"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/cors"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/csrf"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/jwt"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/log"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/meta"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/panic"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/prom"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/ratelimit"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/secure"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/timeout"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/tracing"
	"github.com/rookie-ninja/rk-query"
	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"math"
	"net"
	"net/http"
	"net/http/pprof"
	"path"
	"strconv"
	"strings"
	"sync"
)

// This must be declared in order to register registration function into rk context
// otherwise, rk-boot won't able to bootstrap grpc entry automatically from boot config file
func init() {
	rkentry.RegisterWebFrameRegFunc(RegisterGrpcEntryYAML)
}

const (
	// GrpcEntryType default entry type
	GrpcEntryType = "gRPCEntry"
)

// BootConfig Boot config which is for grpc entry.
type BootConfig struct {
	Grpc []struct {
		Name               string                        `yaml:"name" json:"name"`
		Description        string                        `yaml:"description" json:"description"`
		Port               uint64                        `yaml:"port" json:"port"`
		Enabled            bool                          `yaml:"enabled" json:"enabled"`
		EnableReflection   bool                          `yaml:"enableReflection" json:"enableReflection"`
		NoRecvMsgSizeLimit bool                          `yaml:"noRecvMsgSizeLimit" json:"noRecvMsgSizeLimit"`
		CommonService      rkentry.BootCommonService     `yaml:"commonService" json:"commonService"`
		SW                 rkentry.BootSW                `yaml:"sw" json:"sw"`
		Docs               rkentry.BootDocs              `yaml:"docs" json:"docs"`
		Prom               rkentry.BootProm              `yaml:"prom" json:"prom"`
		Static             rkentry.BootStaticFileHandler `yaml:"static" json:"static"`
		Proxy              BootConfigProxy               `yaml:"proxy" json:"proxy"`
		CertEntry          string                        `yaml:"certEntry" json:"certEntry"`
		LoggerEntry        string                        `yaml:"loggerEntry" json:"loggerEntry"`
		EventEntry         string                        `yaml:"eventEntry" json:"eventEntry"`
		PProf              rkentry.BootPProf             `yaml:"pprof" json:"pprof"`
		EnableRkGwOption   bool                          `yaml:"enableRkGwOption" json:"enableRkGwOption"`
		GwOption           *gwOption                     `yaml:"gwOption" json:"gwOption"`
		Middleware         struct {
			Ignore     []string                `yaml:"ignore" json:"ignore"`
			ErrorModel string                  `yaml:"errorModel" json:"errorModel"`
			Logging    rkmidlog.BootConfig     `yaml:"logging" json:"logging"`
			Prom       rkmidprom.BootConfig    `yaml:"prom" json:"prom"`
			Auth       rkmidauth.BootConfig    `yaml:"auth" json:"auth"`
			Cors       rkmidcors.BootConfig    `yaml:"cors" json:"cors"`
			Secure     rkmidsec.BootConfig     `yaml:"secure" json:"secure"`
			Meta       rkmidmeta.BootConfig    `yaml:"meta" json:"meta"`
			Jwt        rkmidjwt.BootConfig     `yaml:"jwt" json:"jwt"`
			Csrf       rkmidcsrf.BootConfig    `yaml:"csrf" yaml:"csrf"`
			RateLimit  rkmidlimit.BootConfig   `yaml:"rateLimit" json:"rateLimit"`
			Timeout    rkmidtimeout.BootConfig `yaml:"timeout" json:"timeout"`
			Trace      rkmidtrace.BootConfig   `yaml:"trace" json:"trace"`
		} `yaml:"middleware" json:"middleware"`
	} `yaml:"grpc" json:"grpc"`
}

// GrpcEntry implements rkentry.Entry interface.
type GrpcEntry struct {
	entryName         string               `json:"-" yaml:"-"`
	entryType         string               `json:"-" yaml:"-"`
	entryDescription  string               `json:"-" yaml:"-"`
	LoggerEntry       *rkentry.LoggerEntry `json:"-" yaml:"-"`
	EventEntry        *rkentry.EventEntry  `json:"-" yaml:"-"`
	Port              uint64               `json:"-" yaml:"-"`
	TlsConfig         *tls.Config          `json:"-" yaml:"-"`
	TlsConfigInsecure *tls.Config          `json:"-" yaml:"-"`
	// GRPC related
	Server             *grpc.Server                   `json:"-" yaml:"-"`
	ServerOpts         []grpc.ServerOption            `json:"-" yaml:"-"`
	UnaryInterceptors  []grpc.UnaryServerInterceptor  `json:"-" yaml:"-"`
	StreamInterceptors []grpc.StreamServerInterceptor `json:"-" yaml:"-"`
	GrpcRegF           []GrpcRegFunc                  `json:"-" yaml:"-"`
	EnableReflection   bool                           `json:"-" yaml:"-"`
	// Gateway related
	HttpMux         *http.ServeMux             `json:"-" yaml:"-"`
	HttpServer      *http.Server               `json:"-" yaml:"-"`
	GwMux           *gwruntime.ServeMux        `json:"-" yaml:"-"`
	GwMuxOptions    []gwruntime.ServeMuxOption `json:"-" yaml:"-"`
	GwRegF          []GwRegFunc                `json:"-" yaml:"-"`
	GwDialOptions   []grpc.DialOption          `json:"-" yaml:"-"`
	gwCorsOptions   []rkmidcors.Option         `json:"-" yaml:"-"`
	gwSecureOptions []rkmidsec.Option          `json:"-" yaml:"-"`
	gwCsrfOptions   []rkmidcsrf.Option         `json:"-" yaml:"-"`
	// Utility related
	SWEntry            *rkentry.SWEntry                `json:"-" yaml:"-"`
	DocsEntry          *rkentry.DocsEntry              `json:"-" yaml:"-"`
	ProxyEntry         *ProxyEntry                     `json:"-" yaml:"-"`
	PromEntry          *rkentry.PromEntry              `json:"-" yaml:"-"`
	StaticFileEntry    *rkentry.StaticFileHandlerEntry `json:"-" yaml:"-"`
	CommonServiceEntry *rkentry.CommonServiceEntry     `json:"-" yaml:"-"`
	PProfEntry         *rkentry.PProfEntry             `json:"-" yaml:"-"`
	CertEntry          *rkentry.CertEntry              `json:"-" yaml:"-"`
	bootstrapLogOnce   sync.Once                       `json:"-" yaml:"-"`
}

// RegisterGrpcEntryYAML Register grpc entries with provided config file (Must YAML file).
//
// Currently, support two ways to provide config file path.
// 1: With function parameters
// 2: With command line flag "--rkboot" described in rkentry.BootConfigPathFlagKey (Will override function parameter if exists)
// Command line flag has high priority which would override function parameter
//
// Error handling:
// Process will shutdown if any errors occur with rkentry.ShutdownWithError function
//
// Override elements in config file:
// We learned from HELM source code which would override elements in YAML file with "--set" flag followed with comma
// separated key/value pairs.
//
// We are using "--rkset" described in rkentry.BootConfigOverrideKey in order to distinguish with user flags
// Example of common usage: ./binary_file --rkset "key1=val1,key2=val2"
// Example of nested map:   ./binary_file --rkset "outer.inner.key=val"
// Example of slice:        ./binary_file --rkset "outer[0].key=val"
func RegisterGrpcEntryYAML(raw []byte) map[string]rkentry.Entry {
	res := make(map[string]rkentry.Entry)

	// 1: decode config map into boot config struct
	config := &BootConfig{}
	rkentry.UnmarshalBootYAML(raw, config)

	for i := range config.Grpc {
		element := config.Grpc[i]
		if !element.Enabled {
			continue
		}

		// logger entry
		loggerEntry := rkentry.GlobalAppCtx.GetLoggerEntry(element.LoggerEntry)
		if loggerEntry == nil {
			loggerEntry = rkentry.GlobalAppCtx.GetLoggerEntryDefault()
		}

		// event entry
		eventEntry := rkentry.GlobalAppCtx.GetEventEntry(element.EventEntry)
		if eventEntry == nil {
			eventEntry = rkentry.GlobalAppCtx.GetEventEntryDefault()
		}

		// cert entry
		certEntry := rkentry.GlobalAppCtx.GetCertEntry(element.CertEntry)

		// Register swagger entry
		swEntry := rkentry.RegisterSWEntry(&element.SW, rkentry.WithNameSWEntry(element.Name))

		// Register docs entry
		docsEntry := rkentry.RegisterDocsEntry(&element.Docs, rkentry.WithNameDocsEntry(element.Name))

		// Register prometheus entry
		promRegistry := prometheus.NewRegistry()
		promEntry := rkentry.RegisterPromEntry(&element.Prom, rkentry.WithRegistryPromEntry(promRegistry))

		// Register common service entry
		commonServiceEntry := rkentry.RegisterCommonServiceEntry(&element.CommonService)

		// Register static file handler
		staticEntry := rkentry.RegisterStaticFileHandlerEntry(&element.Static, rkentry.WithNameStaticFileHandlerEntry(element.Name))

		// Register pprof entry
		pprofEntry := rkentry.RegisterPProfEntry(&element.PProf, rkentry.WithNamePProfEntry(element.Name))

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
				WithEventEntryProxy(eventEntry),
				WithLoggerEntryProxy(loggerEntry),
				WithRuleProxy(NewRule(opts...)))
		}

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
			WithLoggerEntry(loggerEntry),
			WithEventEntry(eventEntry),
			WithPort(element.Port),
			WithGrpcDialOptions(grpcDialOptions...),
			WithSwEntry(swEntry),
			WithDocsEntry(docsEntry),
			WithPromEntry(promEntry),
			WithProxyEntry(proxy),
			WithGwMuxOptions(gwMuxOpts...),
			WithCommonServiceEntry(commonServiceEntry),
			WithStaticFileHandlerEntry(staticEntry),
			WithCertEntry(certEntry),
			WithPProfEntry(pprofEntry),
			WithEnableReflection(element.EnableReflection),
			WithCertEntry(rkentry.GlobalAppCtx.GetCertEntry(element.CertEntry)))

		// Did we disable message size for receiving?
		if element.NoRecvMsgSizeLimit {
			entry.ServerOpts = append(entry.ServerOpts, grpc.MaxRecvMsgSize(math.MaxInt64))
			entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(math.MaxInt64),
				grpc.MaxCallRecvMsgSize(math.MaxInt64)))
		}

		// add global path ignorance
		rkmid.AddPathToIgnoreGlobal(element.Middleware.Ignore...)

		// set error builder based on error builder
		switch strings.ToLower(element.Middleware.ErrorModel) {
		case "", "google":
			rkmid.SetErrorBuilder(rkerror.NewErrorBuilderGoogle())
		case "amazon":
			rkmid.SetErrorBuilder(rkerror.NewErrorBuilderAMZN())
		}

		// logging middleware
		if element.Middleware.Logging.Enabled {
			entry.AddUnaryInterceptors(rkgrpclog.UnaryServerInterceptor(
				rkmidlog.ToOptions(&element.Middleware.Logging, element.Name, GrpcEntryType,
					loggerEntry, eventEntry)...))
			entry.AddStreamInterceptors(rkgrpclog.StreamServerInterceptor(
				rkmidlog.ToOptions(&element.Middleware.Logging, element.Name, GrpcEntryType,
					loggerEntry, eventEntry)...))
		}

		// Default middleware should be placed after logging middleware, we should make sure interceptors never panic
		// insert panic interceptor
		entry.UnaryInterceptors = append(entry.UnaryInterceptors, rkgrpcpanic.UnaryServerInterceptor(
			rkmidpanic.WithEntryNameAndType(entry.entryName, entry.entryType)))
		entry.StreamInterceptors = append(entry.StreamInterceptors, rkgrpcpanic.StreamServerInterceptor(
			rkmidpanic.WithEntryNameAndType(entry.entryName, entry.entryType)))

		// did we enable metrics interceptor?
		if element.Middleware.Prom.Enabled {
			entry.AddUnaryInterceptors(rkgrpcprom.UnaryServerInterceptor(
				rkmidprom.ToOptions(&element.Middleware.Prom, element.Name, GrpcEntryType,
					promRegistry, rkmidprom.LabelerTypeGrpc)...))
			entry.AddStreamInterceptors(rkgrpcprom.StreamServerInterceptor(
				rkmidprom.ToOptions(&element.Middleware.Prom, element.Name, GrpcEntryType,
					promRegistry, rkmidprom.LabelerTypeGrpc)...))
		}

		// trace middleware
		if element.Middleware.Trace.Enabled {
			entry.AddUnaryInterceptors(rkgrpctrace.UnaryServerInterceptor(
				rkmidtrace.ToOptions(&element.Middleware.Trace, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpctrace.StreamServerInterceptor(
				rkmidtrace.ToOptions(&element.Middleware.Trace, element.Name, GrpcEntryType)...))
		}

		// cors middleware
		if element.Middleware.Cors.Enabled {
			entry.AddGwCorsOptions(rkmidcors.ToOptions(
				&element.Middleware.Cors, element.Name, GrpcEntryType)...)
		}

		// jwt middleware
		if element.Middleware.Jwt.Enabled {
			entry.AddUnaryInterceptors(rkgrpcjwt.UnaryServerInterceptor(
				rkmidjwt.ToOptions(&element.Middleware.Jwt, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpcjwt.StreamServerInterceptor(
				rkmidjwt.ToOptions(&element.Middleware.Jwt, element.Name, GrpcEntryType)...))
		}

		// secure middleware
		if element.Middleware.Secure.Enabled {
			entry.AddGwSecureOptions(rkmidsec.ToOptions(
				&element.Middleware.Secure, element.Name, GrpcEntryType)...)
		}

		// csrf middleware
		if element.Middleware.Csrf.Enabled {
			entry.AddGwCsrfOptions(rkmidcsrf.ToOptions(
				&element.Middleware.Csrf, element.Name, GrpcEntryType)...)
		}

		// meta middleware
		if element.Middleware.Meta.Enabled {
			entry.AddUnaryInterceptors(rkgrpcmeta.UnaryServerInterceptor(
				rkmidmeta.ToOptions(&element.Middleware.Meta, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpcmeta.StreamServerInterceptor(
				rkmidmeta.ToOptions(&element.Middleware.Meta, element.Name, GrpcEntryType)...))
		}

		// auth middleware
		if element.Middleware.Auth.Enabled {
			entry.AddUnaryInterceptors(rkgrpcauth.UnaryServerInterceptor(
				rkmidauth.ToOptions(&element.Middleware.Auth, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpcauth.StreamServerInterceptor(
				rkmidauth.ToOptions(&element.Middleware.Auth, element.Name, GrpcEntryType)...))
		}

		// timeout middleware
		if element.Middleware.Timeout.Enabled {
			entry.AddUnaryInterceptors(rkgrpctimeout.UnaryServerInterceptor(
				rkmidtimeout.ToOptions(&element.Middleware.Timeout, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpctimeout.StreamServerInterceptor(
				rkmidtimeout.ToOptions(&element.Middleware.Timeout, element.Name, GrpcEntryType)...))
		}

		// ratelimit middleware
		if element.Middleware.RateLimit.Enabled {
			entry.AddUnaryInterceptors(rkgrpclimit.UnaryServerInterceptor(
				rkmidlimit.ToOptions(&element.Middleware.RateLimit, element.Name, GrpcEntryType)...))
			entry.AddStreamInterceptors(rkgrpclimit.StreamServerInterceptor(
				rkmidlimit.ToOptions(&element.Middleware.RateLimit, element.Name, GrpcEntryType)...))
		}

		res[element.Name] = entry
	}
	return res
}

// RegisterGrpcEntry Register GrpcEntry with options.
func RegisterGrpcEntry(opts ...GrpcEntryOption) *GrpcEntry {
	entry := &GrpcEntry{
		entryType:        GrpcEntryType,
		entryDescription: "Internal RK entry which helps to bootstrap with Grpc framework.",
		LoggerEntry:      rkentry.GlobalAppCtx.GetLoggerEntryDefault(),
		EventEntry:       rkentry.GlobalAppCtx.GetEventEntryDefault(),
		Port:             8080,
		// gRPC related
		ServerOpts:         make([]grpc.ServerOption, 0),
		UnaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		StreamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		GrpcRegF:           make([]GrpcRegFunc, 0),
		EnableReflection:   true,
		// grpc-gateway related
		GwMuxOptions:    make([]gwruntime.ServeMuxOption, 0),
		GwRegF:          make([]GwRegFunc, 0),
		GwDialOptions:   make([]grpc.DialOption, 0),
		HttpMux:         http.NewServeMux(),
		gwCorsOptions:   make([]rkmidcors.Option, 0),
		gwCsrfOptions:   make([]rkmidcsrf.Option, 0),
		gwSecureOptions: make([]rkmidsec.Option, 0),
	}

	for i := range opts {
		opts[i](entry)
	}

	if len(entry.entryName) < 1 {
		entry.entryName = "grpc-" + strconv.FormatUint(entry.Port, 10)
	}

	// Init TLS config
	if entry.IsTlsEnabled() {
		entry.TlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{*entry.CertEntry.Certificate},
		}
		entry.TlsConfigInsecure = &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{*entry.CertEntry.Certificate},
		}
	}

	// add entry name and entry type into loki syncer if enabled
	entry.LoggerEntry.AddEntryLabelToLokiSyncer(entry)
	entry.EventEntry.AddEntryLabelToLokiSyncer(entry)

	rkentry.GlobalAppCtx.AddEntry(entry)

	return entry
}

// ************* Entry function *************

// GetName Get entry name.
func (entry *GrpcEntry) GetName() string {
	return entry.entryName
}

// GetType Get entry type.
func (entry *GrpcEntry) GetType() string {
	return entry.entryType
}

// String Stringfy entry.
func (entry *GrpcEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// GetDescription Get description of entry.
func (entry *GrpcEntry) GetDescription() string {
	return entry.entryDescription
}

// Bootstrap GrpcEntry.
func (entry *GrpcEntry) Bootstrap(ctx context.Context) {
	event, logger := entry.logBasicInfo("Bootstrap", ctx)

	// 1: Create grpc server
	// 1.1: Make unary and stream interceptors into server opts
	// Important! Do not add tls as options since we already enable tls in listener
	entry.ServerOpts = append(entry.ServerOpts,
		grpc.ChainUnaryInterceptor(entry.UnaryInterceptors...),
		grpc.ChainStreamInterceptor(entry.StreamInterceptors...))

	// 2: Add proxy entry
	if entry.IsProxyEnabled() {
		entry.ServerOpts = append(entry.ServerOpts,
			grpc.ForceServerCodec(Codec()),
			grpc.UnknownServiceHandler(TransparentHandler(entry.ProxyEntry.r.GetDirector())),
		)
		entry.ProxyEntry.Bootstrap(ctx)
	}

	// 3: Create grpc server
	entry.Server = grpc.NewServer(entry.ServerOpts...)

	// 4: Register grpc function into server
	for _, regFunc := range entry.GrpcRegF {
		regFunc(entry.Server)
	}

	// 5: Enable grpc reflection
	if entry.EnableReflection {
		reflection.Register(entry.Server)
	}

	// 6: Create http server based on grpc gateway
	// 6.1: Create gateway mux
	entry.GwMux = gwruntime.NewServeMux(entry.GwMuxOptions...)

	// 7: Inject insecure option into dial option since grpc call is delegated from gateway which is inner code call
	// and which is safe!
	if entry.TlsConfig != nil {
		entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithTransportCredentials(credentials.NewTLS(entry.TlsConfigInsecure)))
	} else {
		entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithInsecure())
	}

	// 8: Register grpc gateway function into GwMux
	for i := range entry.GwRegF {
		err := entry.GwRegF[i](context.Background(), entry.GwMux, "0.0.0.0:"+strconv.FormatUint(entry.Port, 10), entry.GwDialOptions)
		if err != nil {
			entry.EventEntry.FinishWithError(event, err)
			rkentry.ShutdownWithError(err)
		}
	}

	// 9: Make http mux listen on path of / and configure TV, swagger, prometheus path
	entry.HttpMux.Handle("/", entry.GwMux)

	// 10: swagger
	if entry.IsSWEnabled() {
		entry.HttpMux.HandleFunc(entry.SWEntry.Path, entry.SWEntry.ConfigFileHandler())
		entry.SWEntry.Bootstrap(ctx)
	}

	// 11: docs
	if entry.IsDocsEnabled() {
		entry.HttpMux.HandleFunc(entry.DocsEntry.Path, entry.DocsEntry.ConfigFileHandler())
		entry.DocsEntry.Bootstrap(ctx)
	}

	// 12: static file handler
	if entry.IsStaticFileHandlerEnabled() {
		entry.HttpMux.HandleFunc(entry.StaticFileEntry.Path, entry.StaticFileEntry.GetFileHandler())
		entry.StaticFileEntry.Bootstrap(ctx)
	}

	// 13: prometheus
	if entry.IsPromEnabled() {
		// Register prom path into Router.
		entry.HttpMux.Handle(entry.PromEntry.Path, promhttp.HandlerFor(entry.PromEntry.Gatherer, promhttp.HandlerOpts{}))
		entry.PromEntry.Bootstrap(ctx)
	}

	// 14: common service
	if entry.IsCommonServiceEnabled() {
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.ReadyPath, entry.CommonServiceEntry.Ready)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.GcPath, entry.CommonServiceEntry.Gc)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.InfoPath, entry.CommonServiceEntry.Info)
		entry.HttpMux.HandleFunc(entry.CommonServiceEntry.AlivePath, entry.CommonServiceEntry.Alive)

		// Bootstrap common service entry.
		entry.CommonServiceEntry.Bootstrap(ctx)
	}

	// 15: pprof
	if entry.IsPProfEnabled() {
		entry.HttpMux.HandleFunc(entry.PProfEntry.Path, pprof.Index)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "cmdline"), pprof.Cmdline)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "profile"), pprof.Profile)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "symbol"), pprof.Symbol)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "trace"), pprof.Trace)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "allocs"), pprof.Handler("allocs").ServeHTTP)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "block"), pprof.Handler("block").ServeHTTP)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "goroutine"), pprof.Handler("goroutine").ServeHTTP)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "heap"), pprof.Handler("heap").ServeHTTP)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "mutex"), pprof.Handler("mutex").ServeHTTP)
		entry.HttpMux.HandleFunc(path.Join(entry.PProfEntry.Path, "threadcreate"), pprof.Handler("threadcreate").ServeHTTP)
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
	go func(*GrpcEntry) {
		// Create inner listener
		conn, err := net.Listen("tcp4", ":"+strconv.FormatUint(entry.Port, 10))
		if err != nil {
			entry.bootstrapLogOnce.Do(func() {
				entry.EventEntry.FinishWithError(event, err)
			})
			rkentry.ShutdownWithError(err)
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
			httpL := tcpL.Match(cmux.HTTP1Fast("PATCH"))

			// 4: Start both of grpc and http server
			go entry.startGrpcServer(grpcL, logger)
			go entry.startHttpServer(httpL, logger)

			// 5: Start listener
			if err := tcpL.Serve(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				if err != cmux.ErrListenerClosed {
					entry.bootstrapLogOnce.Do(func() {
						entry.EventEntry.FinishWithError(event, err)
					})
					logger.Error("Error occurs while serving TCP listener.", zap.Error(err))
					rkentry.ShutdownWithError(err)
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
					entry.bootstrapLogOnce.Do(func() {
						entry.EventEntry.FinishWithError(event, err)
					})
					logger.Error("Error occurs while serving TLS listener.", zap.Error(err))
					rkentry.ShutdownWithError(err)
				}
			}
		}
	}(entry)

	entry.bootstrapLogOnce.Do(func() {
		// Print link and logging message
		scheme := "http"
		if entry.IsTlsEnabled() {
			scheme = "https"
		}

		if entry.IsSWEnabled() {
			entry.LoggerEntry.Info(fmt.Sprintf("SwaggerEntry: %s://localhost:%d%s", scheme, entry.Port, entry.SWEntry.Path))
		}
		if entry.IsDocsEnabled() {
			entry.LoggerEntry.Info(fmt.Sprintf("DocsEntry: %s://localhost:%d%s", scheme, entry.Port, entry.DocsEntry.Path))
		}
		if entry.IsPromEnabled() {
			entry.LoggerEntry.Info(fmt.Sprintf("PromEntry: %s://localhost:%d%s", scheme, entry.Port, entry.PromEntry.Path))
		}
		if entry.IsStaticFileHandlerEnabled() {
			entry.LoggerEntry.Info(fmt.Sprintf("StaticFileHandlerEntry: %s://localhost:%d%s", scheme, entry.Port, entry.StaticFileEntry.Path))
		}
		if entry.IsCommonServiceEnabled() {
			handlers := []string{
				fmt.Sprintf("%s://localhost:%d%s", scheme, entry.Port, entry.CommonServiceEntry.ReadyPath),
				fmt.Sprintf("%s://localhost:%d%s", scheme, entry.Port, entry.CommonServiceEntry.AlivePath),
				fmt.Sprintf("%s://localhost:%d%s", scheme, entry.Port, entry.CommonServiceEntry.InfoPath),
			}

			entry.LoggerEntry.Info(fmt.Sprintf("CommonSreviceEntry: %s", strings.Join(handlers, ", ")))
		}
		if entry.IsPProfEnabled() {
			entry.LoggerEntry.Info(fmt.Sprintf("PProfEntry: %s://localhost:%d%s", scheme, entry.Port, entry.PProfEntry.Path))
		}
		entry.EventEntry.Finish(event)
	})
}

func (entry *GrpcEntry) startGrpcServer(lis net.Listener, logger *zap.Logger) {
	if err := entry.Server.Serve(lis); err != nil && !strings.Contains(err.Error(), "mux: server closed") {
		logger.Error("Error occurs while serving grpc-server.", zap.Error(err))
		rkentry.ShutdownWithError(err)
	}
}

func (entry *GrpcEntry) startHttpServer(lis net.Listener, logger *zap.Logger) {
	if err := entry.HttpServer.Serve(lis); err != nil && !strings.Contains(err.Error(), "http: Server closed") {
		logger.Error("Error occurs while serving gateway-server.", zap.Error(err))
		rkentry.ShutdownWithError(err)
	}
}

// Interrupt GrpcEntry.
func (entry *GrpcEntry) Interrupt(ctx context.Context) {
	event, logger := entry.logBasicInfo("Interrupt", ctx)

	// Interrupt CommonServiceEntry, SwEntry, TvEntry, PromEntry
	if entry.IsCommonServiceEnabled() {
		entry.CommonServiceEntry.Interrupt(ctx)
	}

	if entry.IsSWEnabled() {
		entry.SWEntry.Interrupt(ctx)
	}

	if entry.IsStaticFileHandlerEnabled() {
		entry.StaticFileEntry.Interrupt(ctx)
	}

	if entry.IsDocsEnabled() {
		entry.DocsEntry.Interrupt(ctx)
	}

	if entry.IsPromEnabled() {
		entry.PromEntry.Interrupt(ctx)
	}

	if entry.IsPProfEnabled() {
		entry.PProfEntry.Interrupt(ctx)
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

	entry.EventEntry.Finish(event)

	rkentry.GlobalAppCtx.RemoveEntry(entry)
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

// AddGwMuxOptions Add mux options at gateway side.
func (entry *GrpcEntry) AddGwMuxOptions(opts ...gwruntime.ServeMuxOption) {
	entry.GwMuxOptions = append(entry.GwMuxOptions, opts...)
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
		"name":                   entry.entryName,
		"type":                   entry.entryType,
		"description":            entry.entryDescription,
		"port":                   entry.Port,
		"swEntry":                entry.SWEntry,
		"docsEntry":              entry.DocsEntry,
		"commonServiceEntry":     entry.CommonServiceEntry,
		"promEntry":              entry.PromEntry,
		"staticFileHandlerEntry": entry.StaticFileEntry,
		"pprofEntry":             entry.PProfEntry,
		"reflection":             entry.EnableReflection,
	}

	if entry.CertEntry != nil {
		m["certEntry"] = entry.CertEntry.GetName()
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Not supported.
func (entry *GrpcEntry) UnmarshalJSON([]byte) error {
	return nil
}

// IsTlsEnabled Is TLS enabled?
func (entry *GrpcEntry) IsTlsEnabled() bool {
	return entry.CertEntry != nil && entry.CertEntry.Certificate != nil
}

// IsCommonServiceEnabled Is common service enabled?
func (entry *GrpcEntry) IsCommonServiceEnabled() bool {
	return entry.CommonServiceEntry != nil
}

// IsProxyEnabled Is proxy enabled?
func (entry *GrpcEntry) IsProxyEnabled() bool {
	return entry.ProxyEntry != nil
}

// IsSWEnabled Is swagger enabled?
func (entry *GrpcEntry) IsSWEnabled() bool {
	return entry.SWEntry != nil
}

// IsPProfEnabled Is pprof enabled?
func (entry *GrpcEntry) IsPProfEnabled() bool {
	return entry.PProfEntry != nil
}

// IsStaticFileHandlerEnabled Is static file handler entry enabled?
func (entry *GrpcEntry) IsStaticFileHandlerEnabled() bool {
	return entry.StaticFileEntry != nil
}

// IsDocsEnabled Is tv enabled?
func (entry *GrpcEntry) IsDocsEnabled() bool {
	return entry.DocsEntry != nil
}

// IsPromEnabled Is prometheus client enabled?
func (entry *GrpcEntry) IsPromEnabled() bool {
	return entry.PromEntry != nil
}

// Add basic fields into event.
func (entry *GrpcEntry) logBasicInfo(operation string, ctx context.Context) (rkquery.Event, *zap.Logger) {
	event := entry.EventEntry.Start(
		operation,
		rkquery.WithEntryName(entry.GetName()),
		rkquery.WithEntryType(entry.GetType()))

	// extract eventId if exists
	if val := ctx.Value("eventId"); val != nil {
		if id, ok := val.(string); ok {
			event.SetEventId(id)
		}
	}

	logger := entry.LoggerEntry.With(
		zap.String("eventId", event.GetEventId()),
		zap.String("entryName", entry.entryName),
		zap.String("entryType", entry.entryType))

	// add general info
	event.AddPayloads(
		zap.Uint64("grpcPort", entry.Port),
		zap.Uint64("gwPort", entry.Port))

	// add SWEntry info
	if entry.IsSWEnabled() {
		event.AddPayloads(
			zap.Bool("swEnabled", true),
			zap.String("swPath", entry.SWEntry.Path))
	}

	// add CommonServiceEntry info
	if entry.IsCommonServiceEnabled() {
		event.AddPayloads(
			zap.Bool("commonServiceEnabled", true))
	}

	// add DocsEntry info
	if entry.IsDocsEnabled() {
		event.AddPayloads(
			zap.Bool("docsEnabled", true),
			zap.String("docsPath", entry.DocsEntry.Path))
	}

	// add pprofEntry info
	if entry.IsPProfEnabled() {
		event.AddPayloads(
			zap.Bool("pprofEnabled", true),
			zap.String("pprofPath", entry.PProfEntry.Path))
	}

	// add PromEntry info
	if entry.IsPromEnabled() {
		event.AddPayloads(
			zap.Bool("promEnabled", true),
			zap.Uint64("promPort", entry.Port),
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

// GetGrpcEntry Get GrpcEntry from rkentry.GlobalAppCtx.
func GetGrpcEntry(name string) *GrpcEntry {
	if raw := rkentry.GlobalAppCtx.GetEntry(GrpcEntryType, name); raw != nil {
		if res, ok := raw.(*GrpcEntry); ok {
			return res
		}
	}

	return nil
}

// *********** Options ***********

// GwRegFunc Registration function grpc gateway.
type GwRegFunc func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error

// GrpcRegFunc Grpc registration func.
type GrpcRegFunc func(server *grpc.Server)

// GrpcEntryOption GrpcEntry option.
type GrpcEntryOption func(*GrpcEntry)

// WithName Provide name.
func WithName(name string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.entryName = name
	}
}

// WithDescription Provide description.
func WithDescription(description string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.entryDescription = description
	}
}

// WithLoggerEntry Provide rkentry.LoggerEntry
func WithLoggerEntry(logger *rkentry.LoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.LoggerEntry = logger
	}
}

// WithEventEntry Provide rkentry.EventEntry
func WithEventEntry(logger *rkentry.EventEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EventEntry = logger
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

// WithCommonServiceEntry Provide rkentry.CommonServiceEntry.
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

// WithSwEntry Provide rkentry.SWEntry.
func WithSwEntry(sw *rkentry.SWEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.SWEntry = sw
	}
}

// WithDocsEntry Provide rkentry.DocsEntry.
func WithDocsEntry(docs *rkentry.DocsEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.DocsEntry = docs
	}
}

// WithPProfEntry Provide rkentry.PProfEntry.
func WithPProfEntry(p *rkentry.PProfEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.PProfEntry = p
	}
}

// WithProxyEntry Provide ProxyEntry.
func WithProxyEntry(proxy *ProxyEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ProxyEntry = proxy
	}
}

// WithPromEntry Provide rkentry.PromEntry.
func WithPromEntry(prom *rkentry.PromEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.PromEntry = prom
	}
}

// WithStaticFileHandlerEntry provide rkentry.StaticFileHandlerEntry.
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
