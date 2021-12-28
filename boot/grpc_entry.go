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
	"github.com/markbates/pkger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/boot/api/third_party/gen/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor/auth"
	rkgrpccors "github.com/rookie-ninja/rk-grpc/interceptor/cors"
	rkgrpccsrf "github.com/rookie-ninja/rk-grpc/interceptor/csrf"
	rkgrpcjwt "github.com/rookie-ninja/rk-grpc/interceptor/jwt"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/meta"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"github.com/rookie-ninja/rk-grpc/interceptor/ratelimit"
	rkgrpcsec "github.com/rookie-ninja/rk-grpc/interceptor/secure"
	"github.com/rookie-ninja/rk-grpc/interceptor/timeout"
	"github.com/rookie-ninja/rk-grpc/interceptor/tracing/telemetry"
	"github.com/rookie-ninja/rk-prom"
	"github.com/rookie-ninja/rk-query"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// This must be declared in order to register registration function into rk context
// otherwise, rk-boot won't able to bootstrap grpc entry automatically from boot config file
func init() {
	rkentry.RegisterEntryRegFunc(RegisterGrpcEntriesWithConfig)
}

const (
	// GrpcEntryType default entry type
	GrpcEntryType = "GrpcEntry"
	// GrpcEntryDescription default entry description
	GrpcEntryDescription = "Internal RK entry which helps to bootstrap with Grpc framework."
)

// GwRegFunc Registration function grpc gateway.
type GwRegFunc func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error

type gwRule struct {
	Method  string `json:"method" yaml:"method"`
	Pattern string `json:"pattern" yaml:"pattern"`
}

// BootConfigGrpc Boot config which is for grpc entry.
//
// 1: Grpc.Name: Name of entry, should be unique globally.
// 2: Grpc.Description: Description of entry.
// 3: Grpc.Enabled: Enable GrpcEntry.
// 4: Grpc.Port: Port of entry.
// 5: Grpc.EnableReflection: Enable gRPC reflection or not.
// 6: Grpc.Cert.Ref: Reference of rkentry.CertEntry.
// 7: Grpc.CommonService.Enabled: Reference of CommonService.
// 8: Grpc.Sw.Enabled: Enable SwEntry.
// 9: Grpc.Sw.Path: Swagger UI path.
// 10: Grpc.Sw.JsonPath: Swagger JSON config file path.
// 11: Grpc.Sw.Headers: Http headers which would be forwarded to user.
// 12: Grpc.Tv.Enabled: Enable TvEntry.
// 13: Grpc.Prom.Pusher.Enabled: Enable prometheus pushgateway pusher.
// 14: Grpc.Prom.Pusher.IntervalMs: Interval in milliseconds while pushing metrics to remote pushGateway.
// 15: Grpc.Prom.Pusher.JobName: Name of pushGateway pusher job.
// 16: Grpc.Prom.Pusher.RemoteAddress: Remote address of pushGateway server.
// 17: Grpc.Prom.Pusher.BasicAuth: Basic auth credential of pushGateway server.
// 18: Grpc.Prom.Pusher.Cert.Ref: Reference of rkentry.CertEntry.
// 19: Grpc.Prom.Cert.Ref: Reference of rkentry.CertEntry.
// 20: Grpc.Interceptors.LoggingZap.Enabled: Enable zap logger interceptor.
// 21: Grpc.Interceptors.LoggingZap.ZapLoggerEncoding: json or console.
// 22: Grpc.Interceptors.LoggingZap.ZapLoggerOutputPaths: Output paths, stdout is supported.
// 23: Grpc.Interceptors.LoggingZap.EventLoggerEncoding: json or console.
// 24: Grpc.Interceptors.LoggingZap.EventLoggerOutputPaths: Output paths, stdout is supported.
// 25: Grpc.Interceptors.MetricsProm.Enabled: Enable prometheus metrics interceptor.
// 26: Grpc.Interceptors.Auth.Enabled: Enable basic auth interceptor.
// 27: Grpc.Interceptors.Auth.Basic: Basic auth credentials as scheme of <user:pass>.
// 28: Grpc.Interceptors.Auth.ApiKey: API key auth type.
// 29: Grpc.Interceptors.Auth.IgnorePrefix: The prefix that ignoring auth.
// 30: Grpc.Interceptors.Meta.Enabled: Meta interceptor which attach meta headers to response.
// 31: Grpc.Interceptors.Meta.Prefix: Meta interceptor which attach meta headers to response with prefix.
// 32: Grpc.Interceptors.Meta.TracingTelemetry.Enabled: Tracing interceptor.
// 33: Grpc.Interceptors.Meta.TracingTelemetry.Exporter.File.Enabled: Tracing interceptor with file as exporter.
// 34: Grpc.Interceptors.Meta.TracingTelemetry.Exporter.File.OutputPath: Exporter output paths.
// 35: Grpc.Interceptors.Meta.TracingTelemetry.Exporter.Jaeger.Enabled: Tracing interceptor with jaeger as exporter.
// 36: Grpc.Interceptors.Meta.TracingTelemetry.Exporter.Jaeger.CollectorEndpoint: Jaeger collector endpoint.
// 37: Grpc.Interceptors.Meta.TracingTelemetry.Exporter.Jaeger.CollectorUsername: Jaeger collector user name.
// 38: Grpc.Interceptors.Meta.TracingTelemetry.Exporter.Jaeger.CollectorPassword: Jaeger collector password.
// 39: Grpc.Interceptors.RateLimit.Enabled: Enable rate limit interceptor.
// 40: Grpc.Interceptors.RateLimit.Algorithm: Algorithm of rate limiter.
// 41: Grpc.Interceptors.RateLimit.ReqPerSec: Request per second.
// 42: Grpc.Interceptors.RateLimit.Paths.Path: Name of gRPC full method.
// 43: Grpc.Interceptors.RateLimit.Paths.ReqPerSec: Request per second by method.
// 44: Grpc.Interceptors.Timeout.Enabled: Enable timeout interceptor.
// 45: Grpc.Interceptors.Timeout.TimeoutMs: Timeout in milliseconds.
// 46: Grpc.Interceptors.Timeout.Paths.path: Name of full path.
// 47: Grpc.Interceptors.Timeout.Paths.TimeoutMs: Timeout in milliseconds by path.
// 48: Grpc.Logger.ZapLogger.Ref: Zap logger reference, see rkentry.ZapLoggerEntry for details.
// 49: Grpc.Logger.EventLogger.Ref: Event logger reference, see rkentry.EventLoggerEntry for details.
// 50: Grpc.GwOption.Marshal.Multiline: Enable multiline in gateway option.
// 51: Grpc.GwOption.Marshal.EmitUnpopulated: Enable emitUnpopulated in gateway option.
// 52: Grpc.GwOption.Marshal.Indent: Set indent in gateway option.
// 53: Grpc.GwOption.Marshal.AllowPartial: Enable allowPartial in gateway option.
// 54: Grpc.GwOption.Marshal.UseProtoNames: Enable useProtoNames in gateway option.
// 55: Grpc.GwOption.Marshal.UseEnumNumbers: Enable useEnumNumbers in gateway option.
// 56: Grpc.GwOption.Unmarshal.AllowPartial: Enable allowPartial in gateway option.
// 57: Grpc.GwOption.Unmarshal.DiscardUnknown: Enable discardUnknown in gateway option.
type BootConfigGrpc struct {
	Grpc []struct {
		Name               string `yaml:"name" json:"name"`
		Description        string `yaml:"description" json:"description"`
		Port               uint64 `yaml:"port" json:"port"`
		Enabled            bool   `yaml:"enabled" json:"enabled"`
		EnableReflection   bool   `yaml:"enableReflection" json:"enableReflection"`
		NoRecvMsgSizeLimit bool   `yaml:"noRecvMsgSizeLimit" json:"noRecvMsgSizeLimit"`
		Cert               struct {
			Ref string `yaml:"ref" json:"ref"`
		} `yaml:"cert" json:"cert"`
		CommonService      BootConfigCommonService `yaml:"commonService" json:"commonService"`
		Sw                 BootConfigSw            `yaml:"sw" json:"sw"`
		Tv                 BootConfigTv            `yaml:"tv" json:"tv"`
		Prom               BootConfigProm          `yaml:"prom" json:"prom"`
		Static             BootConfigStaticHandler `yaml:"static" json:"static"`
		Proxy              BootConfigProxy         `yaml:"proxy" json:"proxy"`
		EnableRkGwOption   bool                    `yaml:"enableRkGwOption" json:"enableRkGwOption"`
		GwOption           *gwOption               `yaml:"gwOption" json:"gwOption"`
		GwMappingFilePaths []string                `yaml:"gwMappingFilePaths" json:"gwMappingFilePaths"`
		Interceptors       struct {
			LoggingZap struct {
				Enabled                bool     `yaml:"enabled" json:"enabled"`
				ZapLoggerEncoding      string   `yaml:"zapLoggerEncoding" json:"zapLoggerEncoding"`
				ZapLoggerOutputPaths   []string `yaml:"zapLoggerOutputPaths" json:"zapLoggerOutputPaths"`
				EventLoggerEncoding    string   `yaml:"eventLoggerEncoding" json:"eventLoggerEncoding"`
				EventLoggerOutputPaths []string `yaml:"eventLoggerOutputPaths" json:"eventLoggerOutputPaths"`
			} `yaml:"loggingZap" json:"loggingZap"`
			MetricsProm struct {
				Enabled bool `yaml:"enabled" json:"enabled"`
			} `yaml:"metricsProm" json:"metricsProm"`
			Auth struct {
				Enabled      bool     `yaml:"enabled" json:"enabled"`
				IgnorePrefix []string `yaml:"ignorePrefix" json:"ignorePrefix"`
				Basic        []string `yaml:"basic" json:"basic"`
				ApiKey       []string `yaml:"apiKey" json:"apiKey"`
			} `yaml:"auth" json:"auth"`
			Cors struct {
				Enabled          bool     `yaml:"enabled" json:"enabled"`
				AllowOrigins     []string `yaml:"allowOrigins" json:"allowOrigins"`
				AllowCredentials bool     `yaml:"allowCredentials" json:"allowCredentials"`
				AllowHeaders     []string `yaml:"allowHeaders" json:"allowHeaders"`
				AllowMethods     []string `yaml:"allowMethods" json:"allowMethods"`
				ExposeHeaders    []string `yaml:"exposeHeaders" json:"exposeHeaders"`
				MaxAge           int      `yaml:"maxAge" json:"maxAge"`
			} `yaml:"cors" json:"cors"`
			Secure struct {
				Enabled               bool     `yaml:"enabled" json:"enabled"`
				IgnorePrefix          []string `yaml:"ignorePrefix" json:"ignorePrefix"`
				XssProtection         string   `yaml:"xssProtection" json:"xssProtection"`
				ContentTypeNosniff    string   `yaml:"contentTypeNosniff" json:"contentTypeNosniff"`
				XFrameOptions         string   `yaml:"xFrameOptions" json:"xFrameOptions"`
				HstsMaxAge            int      `yaml:"hstsMaxAge" json:"hstsMaxAge"`
				HstsExcludeSubdomains bool     `yaml:"hstsExcludeSubdomains" json:"hstsExcludeSubdomains"`
				HstsPreloadEnabled    bool     `yaml:"hstsPreloadEnabled" json:"hstsPreloadEnabled"`
				ContentSecurityPolicy string   `yaml:"contentSecurityPolicy" json:"contentSecurityPolicy"`
				CspReportOnly         bool     `yaml:"cspReportOnly" json:"cspReportOnly"`
				ReferrerPolicy        string   `yaml:"referrerPolicy" json:"referrerPolicy"`
			} `yaml:"secure" json:"secure"`
			Meta struct {
				Enabled bool   `yaml:"enabled" json:"enabled"`
				Prefix  string `yaml:"prefix" json:"prefix"`
			} `yaml:"meta" json:"meta"`
			Jwt struct {
				Enabled      bool     `yaml:"enabled" json:"enabled"`
				IgnorePrefix []string `yaml:"ignorePrefix" json:"ignorePrefix"`
				SigningKey   string   `yaml:"signingKey" json:"signingKey"`
				SigningKeys  []string `yaml:"signingKeys" json:"signingKeys"`
				SigningAlgo  string   `yaml:"signingAlgo" json:"signingAlgo"`
				TokenLookup  string   `yaml:"tokenLookup" json:"tokenLookup"`
				AuthScheme   string   `yaml:"authScheme" json:"authScheme"`
			} `yaml:"jwt" json:"jwt"`
			Csrf struct {
				Enabled        bool     `yaml:"enabled" json:"enabled"`
				IgnorePrefix   []string `yaml:"ignorePrefix" json:"ignorePrefix"`
				TokenLength    int      `yaml:"tokenLength" json:"tokenLength"`
				TokenLookup    string   `yaml:"tokenLookup" json:"tokenLookup"`
				CookieName     string   `yaml:"cookieName" json:"cookieName"`
				CookieDomain   string   `yaml:"cookieDomain" json:"cookieDomain"`
				CookiePath     string   `yaml:"cookiePath" json:"cookiePath"`
				CookieMaxAge   int      `yaml:"cookieMaxAge" json:"cookieMaxAge"`
				CookieHttpOnly bool     `yaml:"cookieHttpOnly" json:"cookieHttpOnly"`
				CookieSameSite string   `yaml:"cookieSameSite" json:"cookieSameSite"`
			} `yaml:"csrf" yaml:"csrf"`
			RateLimit struct {
				Enabled   bool   `yaml:"enabled" json:"enabled"`
				Algorithm string `yaml:"algorithm" json:"algorithm"`
				ReqPerSec int    `yaml:"reqPerSec" json:"reqPerSec"`
				Paths     []struct {
					Path      string `yaml:"path" json:"path"`
					ReqPerSec int    `yaml:"reqPerSec" json:"reqPerSec"`
				} `yaml:"paths" json:"paths"`
			} `yaml:"rateLimit" json:"rateLimit"`
			Timeout struct {
				Enabled   bool `yaml:"enabled" json:"enabled"`
				TimeoutMs int  `yaml:"timeoutMs" json:"timeoutMs"`
				Paths     []struct {
					Path      string `yaml:"path" json:"path"`
					TimeoutMs int    `yaml:"timeoutMs" json:"timeoutMs"`
				} `yaml:"paths" json:"paths"`
			} `yaml:"timeout" json:"timeout"`
			TracingTelemetry struct {
				Enabled  bool `yaml:"enabled" json:"enabled"`
				Exporter struct {
					File struct {
						Enabled    bool   `yaml:"enabled" json:"enabled"`
						OutputPath string `yaml:"outputPath" json:"outputPath"`
					} `yaml:"file" json:"file"`
					Jaeger struct {
						Agent struct {
							Enabled bool   `yaml:"enabled" json:"enabled"`
							Host    string `yaml:"host" json:"host"`
							Port    int    `yaml:"port" json:"port"`
						} `yaml:"agent" json:"agent"`
						Collector struct {
							Enabled  bool   `yaml:"enabled" json:"enabled"`
							Endpoint string `yaml:"endpoint" json:"endpoint"`
							Username string `yaml:"username" json:"username"`
							Password string `yaml:"password" json:"password"`
						} `yaml:"collector" json:"collector"`
					} `yaml:"jaeger" json:"jaeger"`
				} `yaml:"exporter" json:"exporter"`
			} `yaml:"tracingTelemetry" json:"tracingTelemetry"`
		} `yaml:"interceptors" json:"interceptors"`
		Logger struct {
			ZapLogger struct {
				Ref string `yaml:"ref" json:"ref"`
			} `yaml:"zapLogger" json:"zapLogger"`
			EventLogger struct {
				Ref string `yaml:"ref" json:"ref"`
			} `yaml:"eventLogger" json:"eventLogger"`
		} `yaml:"logger" json:"logger"`
	} `yaml:"grpc" json:"grpc"`
}

// GrpcEntry implements rkentry.Entry interface.
//
// 1: EntryName: Name of entry
// 2: EntryType: Type of entry
// 3: EntryDescription: Description of entry
// 4: ZapLoggerEntry: See rkentry.ZapLoggerEntry for details.
// 5: EventLoggerEntry: See rkentry.EventLoggerEntry for details.
// 6: Port: http/https port server listen to.
// 7: TlsConfig: TLS config for http and grpc server
// 8: TlsConfigInsecure: TLS config for grpc client of gateway
// 9: Server: gRPC server created while bootstrapping.
// 10: ServerOpts: Server options for grpc server.
// 11: UnaryInterceptors: Interceptors user enabled.
// 12: StreamInterceptors: Interceptors user enabled.
// 13: GrpcRegF: gRPC registration functions.
// 14: HttpMux: http mux for overall http server
// 15: HttpServer: http server over grpc server
// 16: GwMux: gRPC gateway mux only routes http requests over grpc
// 17: GwMuxOptions: gRPC gateway mux options.
// 18: GwRegF: gRPC gateway registration function which generated from protocol buffer
// 19: GwMappingFilePaths: gRPC gateway to grpc method mapping file paths.
// 20: GwHttpToGrpcMapping: gRPC gateway to grpc method mapping.
// 21: SwEntry: Swagger entry.
// 22: TvEntry: RK tv entry.
// 23: PromEntry: Prometheus client entry.
// 24: CommonServiceEntry: CommonService entry.
// 25: CertEntry: See CertEntry for details.
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
	gwCorsOptions       []rkgrpccors.Option        `json:"-" yaml:"-"`
	gwSecureOptions     []rkgrpcsec.Option         `json:"-" yaml:"-"`
	gwCsrfOptions       []rkgrpccsrf.Option        `json:"-" yaml:"-"`
	// Utility related
	SwEntry            *SwEntry                `json:"-" yaml:"-"`
	TvEntry            *TvEntry                `json:"-" yaml:"-"`
	ProxyEntry         *ProxyEntry             `json:"-" yaml:"-"`
	PromEntry          *PromEntry              `json:"-" yaml:"-"`
	StaticFileEntry    *StaticFileHandlerEntry `json:"-" yaml:"-"`
	CommonServiceEntry *CommonServiceEntry     `json:"-" yaml:"-"`
	CertEntry          *rkentry.CertEntry      `json:"-" yaml:"-"`
}

// GrpcRegFunc Grpc registration func.
type GrpcRegFunc func(server *grpc.Server)

// GrpcEntryOption GrpcEntry option.
type GrpcEntryOption func(*GrpcEntry)

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
	config := &BootConfigGrpc{}
	rkcommon.UnmarshalBootConfig(configFilePath, config)

	for i := range config.Grpc {
		element := config.Grpc[i]
		if !element.Enabled {
			continue
		}

		zapLoggerEntry := rkentry.GlobalAppCtx.GetZapLoggerEntry(element.Logger.ZapLogger.Ref)
		if zapLoggerEntry == nil {
			zapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
		}

		eventLoggerEntry := rkentry.GlobalAppCtx.GetEventLoggerEntry(element.Logger.EventLogger.Ref)
		if eventLoggerEntry == nil {
			eventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
		}

		var commonService *CommonServiceEntry
		// Did we enable common service?
		if element.CommonService.Enabled {
			commonService = NewCommonServiceEntry(
				WithNameCommonService(element.Name),
				WithEventLoggerEntryCommonService(eventLoggerEntry),
				WithZapLoggerEntryCommonService(zapLoggerEntry))
		}

		// Did we enabled swagger?
		var sw *SwEntry
		if element.Sw.Enabled {
			// init swagger custom headers from config
			headers := make(map[string]string, 0)
			for i := range element.Sw.Headers {
				header := element.Sw.Headers[i]
				tokens := strings.Split(header, ":")
				if len(tokens) == 2 {
					headers[tokens[0]] = tokens[1]
				}
			}

			sw = NewSwEntry(
				WithNameSw(element.Name),
				WithPortSw(element.Port),
				WithPathSw(element.Sw.Path),
				WithJsonPathSw(element.Sw.JsonPath),
				WithHeadersSw(headers),
				WithZapLoggerEntrySw(zapLoggerEntry),
				WithEventLoggerEntrySw(eventLoggerEntry),
				WithEnableCommonServiceSw(element.CommonService.Enabled))
		}

		// Did we enable tv?
		var tv *TvEntry
		if element.Tv.Enabled {
			tv = NewTvEntry(
				WithNameTv(element.Name),
				WithEventLoggerEntryTv(eventLoggerEntry),
				WithZapLoggerEntryTv(zapLoggerEntry))
		}

		// Did we enabled static file handler?
		var staticEntry *StaticFileHandlerEntry
		if element.Static.Enabled {
			var fs http.FileSystem
			switch element.Static.SourceType {
			case "pkger":
				fs = pkger.Dir(element.Static.SourcePath)
			case "local":
				if !filepath.IsAbs(element.Static.SourcePath) {
					wd, _ := os.Getwd()
					element.Static.SourcePath = path.Join(wd, element.Static.SourcePath)
				}
				fs = http.Dir(element.Static.SourcePath)
			}

			staticEntry = NewStaticFileHandlerEntry(
				WithZapLoggerEntryStatic(zapLoggerEntry),
				WithEventLoggerEntryStatic(eventLoggerEntry),
				WithNameStatic(fmt.Sprintf("%s-static", element.Name)),
				WithPathStatic(element.Static.Path),
				WithFileSystemStatic(fs))
		}

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

		// Did we enable prom?
		var prom *PromEntry
		var promRegistry *prometheus.Registry
		if element.Prom.Enabled {
			var pusher *rkprom.PushGatewayPusher

			if element.Prom.Pusher.Enabled {
				var certStore *rkentry.CertStore

				if certEntry := rkentry.GlobalAppCtx.GetCertEntry(element.Prom.Pusher.Cert.Ref); certEntry != nil {
					certStore = certEntry.Store
				}

				pusher, _ = rkprom.NewPushGatewayPusher(
					rkprom.WithIntervalMSPusher(time.Duration(element.Prom.Pusher.IntervalMs)*time.Millisecond),
					rkprom.WithRemoteAddressPusher(element.Prom.Pusher.RemoteAddress),
					rkprom.WithJobNamePusher(element.Prom.Pusher.JobName),
					rkprom.WithBasicAuthPusher(element.Prom.Pusher.BasicAuth),
					rkprom.WithZapLoggerEntryPusher(zapLoggerEntry),
					rkprom.WithEventLoggerEntryPusher(eventLoggerEntry),
					rkprom.WithCertStorePusher(certStore))
			}

			promRegistry = prometheus.NewRegistry()
			promRegistry.Register(prometheus.NewGoCollector())
			prom = NewPromEntry(
				WithNameProm(element.Name),
				WithPortProm(element.Port),
				WithPathProm(element.Prom.Path),
				WithZapLoggerEntryProm(zapLoggerEntry),
				WithEventLoggerEntryProm(eventLoggerEntry),
				WithPromRegistryProm(promRegistry),
				WithPusherProm(pusher))
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
			WithNameGrpc(element.Name),
			WithDescriptionGrpc(element.Description),
			WithZapLoggerEntryGrpc(zapLoggerEntry),
			WithEventLoggerEntryGrpc(eventLoggerEntry),
			WithPortGrpc(element.Port),
			WithGrpcDialOptionsGrpc(grpcDialOptions...),
			WithSwEntryGrpc(sw),
			WithTvEntryGrpc(tv),
			WithPromEntryGrpc(prom),
			WithProxyEntryGrpc(proxy),
			WithGwMuxOptionsGrpc(gwMuxOpts...),
			WithCommonServiceEntryGrpc(commonService),
			WithStaticFileHandlerEntryGrpc(staticEntry),
			WithEnableReflectionGrpc(element.EnableReflection),
			WithGwMappingFilePathsGrpc(element.GwMappingFilePaths...),
			WithCertEntryGrpc(rkentry.GlobalAppCtx.GetCertEntry(element.Cert.Ref)))

		// Did we disabled message size for receiving?
		if element.NoRecvMsgSizeLimit {
			entry.ServerOpts = append(entry.ServerOpts, grpc.MaxRecvMsgSize(math.MaxInt64))
			entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(math.MaxInt64),
				grpc.MaxCallRecvMsgSize(math.MaxInt64)))
		}

		// did we enabled logging interceptor?
		if element.Interceptors.LoggingZap.Enabled {
			opts := make([]rkgrpclog.Option, 0)
			opts = append(opts,
				rkgrpclog.WithEventLoggerEntry(eventLoggerEntry),
				rkgrpclog.WithZapLoggerEntry(zapLoggerEntry),
				rkgrpclog.WithEntryNameAndType(element.Name, GrpcEntryType))

			if strings.ToLower(element.Interceptors.LoggingZap.ZapLoggerEncoding) == "json" {
				opts = append(opts, rkgrpclog.WithZapLoggerEncoding(rkgrpclog.ENCODING_JSON))
			}

			if strings.ToLower(element.Interceptors.LoggingZap.EventLoggerEncoding) == "json" {
				opts = append(opts, rkgrpclog.WithEventLoggerEncoding(rkgrpclog.ENCODING_JSON))
			}

			if len(element.Interceptors.LoggingZap.ZapLoggerOutputPaths) > 0 {
				opts = append(opts, rkgrpclog.WithZapLoggerOutputPaths(element.Interceptors.LoggingZap.ZapLoggerOutputPaths...))
			}

			if len(element.Interceptors.LoggingZap.EventLoggerOutputPaths) > 0 {
				opts = append(opts, rkgrpclog.WithEventLoggerOutputPaths(element.Interceptors.LoggingZap.EventLoggerOutputPaths...))
			}

			entry.AddUnaryInterceptors(rkgrpclog.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpclog.StreamServerInterceptor(opts...))
		}

		// did we enabled metrics interceptor?
		if element.Interceptors.MetricsProm.Enabled {
			opts := make([]rkgrpcmetrics.Option, 0)
			opts = append(opts,
				rkgrpcmetrics.WithRegisterer(promRegistry),
				rkgrpcmetrics.WithEntryNameAndType(element.Name, GrpcEntryType))

			entry.AddUnaryInterceptors(rkgrpcmetrics.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpcmetrics.StreamServerInterceptor(opts...))
		}

		// did we enabled tracing interceptor?
		if element.Interceptors.TracingTelemetry.Enabled {
			var exporter trace.SpanExporter

			if element.Interceptors.TracingTelemetry.Exporter.File.Enabled {
				exporter = rkgrpctrace.CreateFileExporter(element.Interceptors.TracingTelemetry.Exporter.File.OutputPath)
			}

			if element.Interceptors.TracingTelemetry.Exporter.Jaeger.Agent.Enabled {
				opts := make([]jaeger.AgentEndpointOption, 0)
				if len(element.Interceptors.TracingTelemetry.Exporter.Jaeger.Agent.Host) > 0 {
					opts = append(opts,
						jaeger.WithAgentHost(element.Interceptors.TracingTelemetry.Exporter.Jaeger.Agent.Host))
				}
				if element.Interceptors.TracingTelemetry.Exporter.Jaeger.Agent.Port > 0 {
					opts = append(opts,
						jaeger.WithAgentPort(
							fmt.Sprintf("%d", element.Interceptors.TracingTelemetry.Exporter.Jaeger.Agent.Port)))
				}

				exporter = rkgrpctrace.CreateJaegerExporter(jaeger.WithAgentEndpoint(opts...))
			}

			if element.Interceptors.TracingTelemetry.Exporter.Jaeger.Collector.Enabled {
				opts := []jaeger.CollectorEndpointOption{
					jaeger.WithUsername(element.Interceptors.TracingTelemetry.Exporter.Jaeger.Collector.Username),
					jaeger.WithPassword(element.Interceptors.TracingTelemetry.Exporter.Jaeger.Collector.Password),
				}

				if len(element.Interceptors.TracingTelemetry.Exporter.Jaeger.Collector.Endpoint) > 0 {
					opts = append(opts, jaeger.WithEndpoint(element.Interceptors.TracingTelemetry.Exporter.Jaeger.Collector.Endpoint))
				}

				exporter = rkgrpctrace.CreateJaegerExporter(jaeger.WithCollectorEndpoint(opts...))
			}

			opts := []rkgrpctrace.Option{
				rkgrpctrace.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpctrace.WithExporter(exporter),
			}

			entry.AddUnaryInterceptors(rkgrpctrace.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpctrace.StreamServerInterceptor(opts...))
		}

		// did we enabled jwt interceptor?
		if element.Interceptors.Jwt.Enabled {
			var signingKey []byte
			if len(element.Interceptors.Jwt.SigningKey) > 0 {
				signingKey = []byte(element.Interceptors.Jwt.SigningKey)
			}

			opts := []rkgrpcjwt.Option{
				rkgrpcjwt.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpcjwt.WithSigningKey(signingKey),
				rkgrpcjwt.WithSigningAlgorithm(element.Interceptors.Jwt.SigningAlgo),
				rkgrpcjwt.WithTokenLookup(element.Interceptors.Jwt.TokenLookup),
				rkgrpcjwt.WithAuthScheme(element.Interceptors.Jwt.AuthScheme),
				rkgrpcjwt.WithIgnorePrefix(element.Interceptors.Jwt.IgnorePrefix...),
			}

			for _, v := range element.Interceptors.Jwt.SigningKeys {
				tokens := strings.SplitN(v, ":", 2)
				if len(tokens) == 2 {
					opts = append(opts, rkgrpcjwt.WithSigningKeys(tokens[0], tokens[1]))
				}
			}

			entry.AddUnaryInterceptors(rkgrpcjwt.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpcjwt.StreamServerInterceptor(opts...))
		}

		// did we enabled secure interceptor?
		// secure interceptor is for grpc-gateway
		if element.Interceptors.Secure.Enabled {
			opts := []rkgrpcsec.Option{
				rkgrpcsec.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpcsec.WithXSSProtection(element.Interceptors.Secure.XssProtection),
				rkgrpcsec.WithContentTypeNosniff(element.Interceptors.Secure.ContentTypeNosniff),
				rkgrpcsec.WithXFrameOptions(element.Interceptors.Secure.XFrameOptions),
				rkgrpcsec.WithHSTSMaxAge(element.Interceptors.Secure.HstsMaxAge),
				rkgrpcsec.WithHSTSExcludeSubdomains(element.Interceptors.Secure.HstsExcludeSubdomains),
				rkgrpcsec.WithHSTSPreloadEnabled(element.Interceptors.Secure.HstsPreloadEnabled),
				rkgrpcsec.WithContentSecurityPolicy(element.Interceptors.Secure.ContentSecurityPolicy),
				rkgrpcsec.WithCSPReportOnly(element.Interceptors.Secure.CspReportOnly),
				rkgrpcsec.WithReferrerPolicy(element.Interceptors.Secure.ReferrerPolicy),
				rkgrpcsec.WithIgnorePrefix(element.Interceptors.Secure.IgnorePrefix...),
			}

			entry.AddGwSecureOptions(opts...)
		}

		// did we enabled csrf interceptor?
		// CSRF interceptor is for grpc-gateway
		if element.Interceptors.Csrf.Enabled {
			opts := []rkgrpccsrf.Option{
				rkgrpccsrf.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpccsrf.WithTokenLength(element.Interceptors.Csrf.TokenLength),
				rkgrpccsrf.WithTokenLookup(element.Interceptors.Csrf.TokenLookup),
				rkgrpccsrf.WithCookieName(element.Interceptors.Csrf.CookieName),
				rkgrpccsrf.WithCookieDomain(element.Interceptors.Csrf.CookieDomain),
				rkgrpccsrf.WithCookiePath(element.Interceptors.Csrf.CookiePath),
				rkgrpccsrf.WithCookieMaxAge(element.Interceptors.Csrf.CookieMaxAge),
				rkgrpccsrf.WithCookieHTTPOnly(element.Interceptors.Csrf.CookieHttpOnly),
				rkgrpccsrf.WithCookieSameSite(element.Interceptors.Csrf.CookieSameSite),
				rkgrpccsrf.WithIgnorePrefix(element.Interceptors.Csrf.IgnorePrefix...),
			}

			entry.AddGwCsrfOptions(opts...)
		}

		// did we enabled cors interceptor?
		// CORS interceptor is for grpc-gateway
		if element.Interceptors.Cors.Enabled {
			opts := []rkgrpccors.Option{
				rkgrpccors.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpccors.WithAllowOrigins(element.Interceptors.Cors.AllowOrigins...),
				rkgrpccors.WithAllowCredentials(element.Interceptors.Cors.AllowCredentials),
				rkgrpccors.WithExposeHeaders(element.Interceptors.Cors.ExposeHeaders...),
				rkgrpccors.WithMaxAge(element.Interceptors.Cors.MaxAge),
				rkgrpccors.WithAllowHeaders(element.Interceptors.Cors.AllowHeaders...),
				rkgrpccors.WithAllowMethods(element.Interceptors.Cors.AllowMethods...),
			}

			entry.AddGwCorsOptions(opts...)
		}

		// did we enabled meta interceptor?
		if element.Interceptors.Meta.Enabled {
			opts := []rkgrpcmeta.Option{
				rkgrpcmeta.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpcmeta.WithPrefix(element.Interceptors.Meta.Prefix),
			}

			entry.AddUnaryInterceptors(rkgrpcmeta.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpcmeta.StreamServerInterceptor(opts...))
		}

		// did we enabled auth interceptor?
		if element.Interceptors.Auth.Enabled {
			opts := make([]rkgrpcauth.Option, 0)
			opts = append(opts,
				rkgrpcauth.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpcauth.WithBasicAuth(element.Interceptors.Auth.Basic...),
				rkgrpcauth.WithApiKeyAuth(element.Interceptors.Auth.ApiKey...))

			opts = append(opts, rkgrpcauth.WithIgnorePrefix(element.Interceptors.Auth.IgnorePrefix...))

			entry.AddUnaryInterceptors(rkgrpcauth.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpcauth.StreamServerInterceptor(opts...))
		}

		// did we enabled timeout interceptor?
		// This should be in front of rate limit interceptor since rate limit may block over the threshold of timeout.
		if element.Interceptors.Timeout.Enabled {
			opts := make([]rkgrpctimeout.Option, 0)
			opts = append(opts,
				rkgrpctimeout.WithEntryNameAndType(element.Name, GrpcEntryType))

			timeout := time.Duration(element.Interceptors.Timeout.TimeoutMs) * time.Millisecond
			opts = append(opts, rkgrpctimeout.WithTimeoutAndResp(timeout, nil))

			for i := range element.Interceptors.Timeout.Paths {
				e := element.Interceptors.Timeout.Paths[i]
				timeout := time.Duration(e.TimeoutMs) * time.Millisecond
				opts = append(opts, rkgrpctimeout.WithTimeoutAndRespByPath(e.Path, timeout, nil))
			}

			entry.AddUnaryInterceptors(rkgrpctimeout.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpctimeout.StreamServerInterceptor(opts...))
		}

		// did we enabled rate limit interceptor?
		if element.Interceptors.RateLimit.Enabled {
			opts := make([]rkgrpclimit.Option, 0)
			opts = append(opts, rkgrpclimit.WithEntryNameAndType(element.Name, GrpcEntryType))

			if len(element.Interceptors.RateLimit.Algorithm) > 0 {
				opts = append(opts, rkgrpclimit.WithAlgorithm(element.Interceptors.RateLimit.Algorithm))
			}
			opts = append(opts, rkgrpclimit.WithReqPerSec(element.Interceptors.RateLimit.ReqPerSec))

			for i := range element.Interceptors.RateLimit.Paths {
				e := element.Interceptors.RateLimit.Paths[i]
				opts = append(opts, rkgrpclimit.WithReqPerSecByPath(e.Path, e.ReqPerSec))
			}

			entry.AddUnaryInterceptors(rkgrpclimit.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpclimit.StreamServerInterceptor(opts...))
		}

		res[element.Name] = entry
	}
	return res
}

// WithNameGrpc Provide name.
func WithNameGrpc(name string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EntryName = name
	}
}

// WithDescriptionGrpc Provide description.
func WithDescriptionGrpc(description string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EntryDescription = description
	}
}

// WithZapLoggerEntryGrpc Provide rkentry.ZapLoggerEntry
func WithZapLoggerEntryGrpc(logger *rkentry.ZapLoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ZapLoggerEntry = logger
	}
}

// WithEventLoggerEntryGrpc Provide rkentry.EventLoggerEntry
func WithEventLoggerEntryGrpc(logger *rkentry.EventLoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EventLoggerEntry = logger
	}
}

// WithPortGrpc Provide port.
func WithPortGrpc(port uint64) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.Port = port
	}
}

// WithServerOptionsGrpc Provide grpc.ServerOption.
func WithServerOptionsGrpc(opts ...grpc.ServerOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ServerOpts = append(entry.ServerOpts, opts...)
	}
}

// WithUnaryInterceptorsGrpc Provide grpc.UnaryServerInterceptor.
func WithUnaryInterceptorsGrpc(opts ...grpc.UnaryServerInterceptor) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.UnaryInterceptors = append(entry.UnaryInterceptors, opts...)
	}
}

// WithStreamInterceptorsGrpc Provide grpc.StreamServerInterceptor.
func WithStreamInterceptorsGrpc(opts ...grpc.StreamServerInterceptor) GrpcEntryOption {
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

// WithCertEntryGrpc Provide rkentry.CertEntry.
func WithCertEntryGrpc(certEntry *rkentry.CertEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.CertEntry = certEntry
	}
}

// WithCommonServiceEntryGrpc Provide CommonServiceEntry.
func WithCommonServiceEntryGrpc(commonService *CommonServiceEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.CommonServiceEntry = commonService
	}
}

// WithEnableReflectionGrpc Provide EnableReflection.
func WithEnableReflectionGrpc(enabled bool) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EnableReflection = enabled
	}
}

// WithSwEntryGrpc Provide SwEntry.
func WithSwEntryGrpc(sw *SwEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.SwEntry = sw
	}
}

// WithTvEntryGrpc Provide TvEntry.
func WithTvEntryGrpc(tv *TvEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.TvEntry = tv
	}
}

// WithProxyEntryGrpc Provide ProxyEntry.
func WithProxyEntryGrpc(proxy *ProxyEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ProxyEntry = proxy
	}
}

// WithPromEntryGrpc Provide PromEntry.
func WithPromEntryGrpc(prom *PromEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.PromEntry = prom
	}
}

// WithStaticFileHandlerEntryGrpc provide StaticFileHandlerEntry.
func WithStaticFileHandlerEntryGrpc(staticEntry *StaticFileHandlerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.StaticFileEntry = staticEntry
	}
}

// WithGwRegFGrpc Provide registration function.
func WithGwRegFGrpc(f ...GwRegFunc) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwRegF = append(entry.GwRegF, f...)
	}
}

// WithGrpcDialOptionsGrpc Provide grpc dial options.
func WithGrpcDialOptionsGrpc(opts ...grpc.DialOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwDialOptions = append(entry.GwDialOptions, opts...)
	}
}

// WithGwMuxOptionsGrpc Provide gateway server mux options.
func WithGwMuxOptionsGrpc(opts ...gwruntime.ServeMuxOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwMuxOptions = append(entry.GwMuxOptions, opts...)
	}
}

// WithGwMappingFilePathsGrpc Provide gateway mapping configuration file paths.
func WithGwMappingFilePathsGrpc(paths ...string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwMappingFilePaths = append(entry.GwMappingFilePaths, paths...)
	}
}

// RegisterGrpcEntry Register GrpcEntry with options.
func RegisterGrpcEntry(opts ...GrpcEntryOption) *GrpcEntry {
	entry := &GrpcEntry{
		EntryType:        GrpcEntryType,
		EntryDescription: GrpcEntryDescription,
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		Port:             8080,
		// GRPC related
		ServerOpts:         make([]grpc.ServerOption, 0),
		UnaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		StreamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		GrpcRegF:           make([]GrpcRegFunc, 0),
		EnableReflection:   true,
		// Gateway related
		GwMuxOptions:        make([]gwruntime.ServeMuxOption, 0),
		GwRegF:              make([]GwRegFunc, 0),
		GwMappingFilePaths:  make([]string, 0),
		GwHttpToGrpcMapping: make(map[string]*gwRule),
		GwDialOptions:       make([]grpc.DialOption, 0),
		HttpMux:             http.NewServeMux(),
		gwCorsOptions:       make([]rkgrpccors.Option, 0),
	}

	for i := range opts {
		opts[i](entry)
	}

	// The ideal interceptor sequence would be like bellow

	//    +-------+
	//    |  log  |
	//    +-------+
	//        |
	//    +-------+
	//    | prom  |
	//    +-------+
	//        |
	//   +---------+
	//   | tracing |
	//   +---------+
	//        |
	//    +------+
	//    | meta |
	//    +------+
	//        |
	//    +-------+
	//    | auth  |
	//    +-------+
	//        |
	//    +---------+
	//    | timeout |
	//    +---------+
	//        |
	//    +-------------+
	//    | rate limit  |
	//    +-------------+
	//        |
	//    +-------+
	//    | panic |
	//    +-------+
	// Append panic interceptor at the end
	entry.UnaryInterceptors = append(entry.UnaryInterceptors, rkgrpcpanic.UnaryServerInterceptor(
		rkgrpcpanic.WithEntryNameAndType(entry.EntryName, entry.EntryType)))
	entry.StreamInterceptors = append(entry.StreamInterceptors, rkgrpcpanic.StreamServerInterceptor(
		rkgrpcpanic.WithEntryNameAndType(entry.EntryName, entry.EntryType)))

	if entry.ZapLoggerEntry == nil {
		entry.ZapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
	}

	if entry.EventLoggerEntry == nil {
		entry.EventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = "GrpcServer-" + strconv.FormatUint(entry.Port, 10)
	}

	// Register common service into grpc and grpc gateway
	if entry.CommonServiceEntry != nil {
		entry.GrpcRegF = append(entry.GrpcRegF, entry.CommonServiceEntry.GrpcRegF)
		entry.GwRegF = append(entry.GwRegF, entry.CommonServiceEntry.GwRegF)
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

	rkentry.GlobalAppCtx.AddEntry(entry)

	return entry
}

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
func (entry *GrpcEntry) AddGwCorsOptions(opts ...rkgrpccors.Option) {
	entry.gwCorsOptions = append(entry.gwCorsOptions, opts...)
}

// AddGwCsrfOptions Enable CORS at gateway side with options.
func (entry *GrpcEntry) AddGwCsrfOptions(opts ...rkgrpccsrf.Option) {
	entry.gwCsrfOptions = append(entry.gwCsrfOptions, opts...)
}

// AddGwSecureOptions Enable secure at gateway side with options.
func (entry *GrpcEntry) AddGwSecureOptions(opts ...rkgrpcsec.Option) {
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

	// 3: Parse gateway mapping file paths, this will record http to grpc path map into a map
	// which will be used for /apis call in CommonServiceEntry
	entry.parseGwMapping()

	// 4: Create grpc server
	// 4.1: Make unary and stream interceptors into server opts
	// Important! Do not add tls as options since we already enable tls in listener
	entry.ServerOpts = append(entry.ServerOpts,
		grpc.ChainUnaryInterceptor(entry.UnaryInterceptors...),
		grpc.ChainStreamInterceptor(entry.StreamInterceptors...))

	// 4.2: Add proxy entry
	if entry.IsProxyEnabled() {
		entry.ServerOpts = append(entry.ServerOpts,
			grpc.ForceServerCodec(Codec()),
			grpc.UnknownServiceHandler(TransparentHandler(entry.ProxyEntry.r.GetDirector())),
		)
		entry.ProxyEntry.Bootstrap(ctx)
	}

	// 4.3: Create grpc server
	entry.Server = grpc.NewServer(entry.ServerOpts...)
	// 4.4: Register grpc function into server
	for _, regFunc := range entry.GrpcRegF {
		regFunc(entry.Server)
	}
	// 4.5: Enable grpc reflection
	if entry.EnableReflection {
		reflection.Register(entry.Server)
	}

	// 5: Create http server based on grpc gateway
	// 5.1: Create gateway mux
	entry.GwMux = gwruntime.NewServeMux(entry.GwMuxOptions...)
	// 5.2: Inject insecure option into dial option since grpc call is delegated from gateway which is inner code call
	// and which is safe!
	if entry.TlsConfig != nil {
		entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithTransportCredentials(credentials.NewTLS(entry.TlsConfigInsecure)))
	} else {
		entry.GwDialOptions = append(entry.GwDialOptions, grpc.WithInsecure())
	}
	// 5.3: Register grpc gateway function into GwMux
	for i := range entry.GwRegF {
		err := entry.GwRegF[i](context.Background(), entry.GwMux, "0.0.0.0:"+strconv.FormatUint(entry.Port, 10), entry.GwDialOptions)
		if err != nil {
			entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
			rkcommon.ShutdownWithError(err)
		}
	}
	// 5.4: Make http mux listen on path of / and configure TV, swagger, prometheus path
	entry.HttpMux.Handle("/", entry.GwMux)
	if entry.IsTvEnabled() {
		entry.HttpMux.HandleFunc("/rk/v1/tv/", entry.TvEntry.TV)
		entry.HttpMux.HandleFunc("/rk/v1/assets/tv/", entry.TvEntry.AssetsFileHandler)
	}
	if entry.IsSwEnabled() {
		entry.HttpMux.HandleFunc(entry.SwEntry.Path, entry.SwEntry.ConfigFileHandler)
		entry.HttpMux.HandleFunc("/rk/v1/assets/sw/", entry.SwEntry.AssetsFileHandler)
	}
	if entry.IsStaticFileHandlerEnabled() {
		entry.HttpMux.HandleFunc(entry.StaticFileEntry.Path, entry.StaticFileEntry.GetFileHandler)
	}
	if entry.IsPromEnabled() {
		// Register prom path into Router.
		entry.HttpMux.Handle(entry.PromEntry.Path, promhttp.HandlerFor(entry.PromEntry.Gatherer, promhttp.HandlerOpts{}))
	}
	// 5.5: Create http server
	var httpHandler http.Handler
	httpHandler = entry.HttpMux

	// 5.6: If CORS enabled, then add interceptor for grpc-gateway
	if len(entry.gwCorsOptions) > 0 {
		httpHandler = rkgrpccors.Interceptor(httpHandler, entry.gwCorsOptions...)
	}

	// 5.7: If Secure enabled, then add interceptor for grpc-gateway
	if len(entry.gwSecureOptions) > 0 {
		httpHandler = rkgrpcsec.Interceptor(httpHandler, entry.gwSecureOptions...)
	}

	// 5.8: If CSRF enabled, then add interceptor for grpc-gateway
	if len(entry.gwCsrfOptions) > 0 {
		httpHandler = rkgrpccsrf.Interceptor(httpHandler, entry.gwCsrfOptions...)
	}

	entry.HttpServer = &http.Server{
		Addr:    "0.0.0.0:" + strconv.FormatUint(entry.Port, 10),
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}),
	}

	// 6: Bootstrap CommonServiceEntry, SwEntry, PromEntry and TvEntry
	if entry.IsCommonServiceEnabled() {
		entry.CommonServiceEntry.Bootstrap(ctx)
	}
	if entry.IsSwEnabled() {
		entry.SwEntry.Bootstrap(ctx)
	}
	if entry.IsPromEnabled() {
		entry.PromEntry.Bootstrap(ctx)
	}
	if entry.IsStaticFileHandlerEnabled() {
		entry.StaticFileEntry.Bootstrap(ctx)
	}
	if entry.IsTvEnabled() {
		entry.TvEntry.Bootstrap(ctx)
	}

	// 7: Start http server
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
	// Parse common service if common service is enabled and GwMappingFilePath is not empty.
	if entry.IsCommonServiceEnabled() && len(entry.CommonServiceEntry.GwMappingFilePath) > 0 {
		bytes := readFileFromPkger(entry.CommonServiceEntry.GwMappingFilePath)
		entry.parseGwMappingHelper(bytes)
	}

	// Parse user services.
	for i := range entry.GwMappingFilePaths {
		filePath := entry.GwMappingFilePaths[i]

		if len(filePath) < 1 {
			continue
		}

		// Deal with relative directory.
		if !path.IsAbs(filePath) {
			if wd, err := os.Getwd(); err != nil {
				entry.ZapLoggerEntry.GetLogger().Warn("Failed to get working directory.", zap.Error(err))
				continue
			} else {
				filePath = path.Join(wd, filePath)
			}
		}

		// Read file and parse mapping
		if bytes, err := ioutil.ReadFile(filePath); err != nil {
			entry.ZapLoggerEntry.GetLogger().Warn("Failed to read file.", zap.Error(err))
			continue
		} else {
			entry.parseGwMappingHelper(bytes)
		}
	}
}

// Helper function of parseGwMapping
func (entry *GrpcEntry) parseGwMappingHelper(bytes []byte) {
	if len(bytes) < 1 {
		return
	}

	mapping := &rk_grpc_common_v1.GrpcAPIService{}

	jsonContents, err := yaml.YAMLToJSON(bytes)
	if err != nil {
		entry.ZapLoggerEntry.GetLogger().Warn("Failed to convert grpc api config.", zap.Error(err))
	}

	// GrpcAPIService is incomplete, accept unknown fields.
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	if err := unmarshaler.Unmarshal(jsonContents, mapping); err != nil {
		entry.ZapLoggerEntry.GetLogger().Warn("Failed to parse grpc api config.", zap.Error(err))
	}

	rules := mapping.GetHttp().GetRules()

	for i := range rules {
		element := rules[i]
		rule := &gwRule{}
		entry.GwHttpToGrpcMapping[element.GetSelector()] = rule
		// Iterate all possible mappings, we are tracking GET, PUT, POST, PATCH, DELETE only.
		if len(element.GetGet()) > 0 {
			rule.Pattern = strings.TrimSuffix(element.GetGet(), "/")
			rule.Method = "GET"
		} else if len(element.GetPut()) > 0 {
			rule.Pattern = strings.TrimSuffix(element.GetPut(), "/")
			rule.Method = "PUT"
		} else if len(element.GetPost()) > 0 {
			rule.Pattern = strings.TrimSuffix(element.GetPost(), "/")
			rule.Method = "POST"
		} else if len(element.GetDelete()) > 0 {
			rule.Pattern = strings.TrimSuffix(element.GetDelete(), "/")
			rule.Method = "DELETE"
		} else if len(element.GetPatch()) > 0 {
			rule.Pattern = strings.TrimSuffix(element.GetPatch(), "/")
			rule.Method = "PATCH"
		}
	}
}

// GetGrpcEntry Get GrpcEntry from rkentry.GlobalAppCtx.
func GetGrpcEntry(name string) *GrpcEntry {
	entryRaw := rkentry.GlobalAppCtx.GetEntry(name)
	if entryRaw == nil {
		return nil
	}

	entry, _ := entryRaw.(*GrpcEntry)
	return entry
}
