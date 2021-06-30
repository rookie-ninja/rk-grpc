// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/auth"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/meta"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"github.com/rookie-ninja/rk-grpc/interceptor/tracing/telemetry"
	"github.com/rookie-ninja/rk-prom"
	"github.com/rookie-ninja/rk-query"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"net"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	GrpcEntryType        = "GrpcEntry"
	GrpcEntryDescription = "Internal RK entry which helps to bootstrap with Grpc framework."
	bootstrapEventIdKey  = "bootstrapEventId"
)

// This must be declared in order to register registration function into rk context
// otherwise, rk-boot won't able to bootstrap grpc entry automatically from boot config file
func init() {
	rkentry.RegisterEntryRegFunc(RegisterGrpcEntriesWithConfig)
}

// Boot config which is for grpc entry.
//
// 1: Grpc.Name: Name of entry, should be unique globally.
// 2: Grpc.Description: Description of entry.
// 3: Grpc.Port: Port of entry.
// 4: Grpc.Cert.Ref: Reference of rkentry.CertEntry.
// 5: Grpc.GW.Enabled: Enable grpc gateway.
// 6: Grpc.GW.Port: Grpc gateway port.
// 7: Grpc.GW.Cert.Ref: Reference of rkentry.CertEntry.
// 8: Grpc.GW.Logger.ZapLogger.Ref: Reference of rkentry.ZapLoggerEntry.
// 9: Grpc.GW.Logger.EventLogger.Ref: Reference of rkentry.EventLoggerEntry.
// 10: Grpc.GW.PathPrefix: Grpc gateway path prefix for all path.
// 11: Grpc.Gw.Tv.Enabled: Enable TvEntry.
// 12: Grpc.GW.Sw.Enabled: Enable SwEntry.
// 13: Grpc.GW.Sw.Path: Swagger UI path.
// 14: Grpc.GW.Sw.JsonPath: Swagger JSON config file path.
// 15: Grpc.GW.Sw.Headers: Http headers which would be forwarded to user.
// 16: Grpc.GW.Prom.Pusher.Enabled: Enable prometheus pushgateway pusher.
// 17: Grpc.GW.Prom.Pusher.IntervalMs: Interval in milliseconds while pushing metrics to remote pushGateway.
// 18: Grpc.GW.Prom.Pusher.JobName: Name of pushGateway pusher job.
// 19: Grpc.GW.Prom.Pusher.RemoteAddress: Remote address of pushGateway server.
// 20: Grpc.GW.Prom.Pusher.BasicAuth: Basic auth credential of pushGateway server.
// 21: Grpc.GW.Prom.Pusher.Cert.Ref: Reference of rkentry.CertEntry.
// 22: Grpc.GW.Prom.Cert.Ref: Reference of rkentry.CertEntry.
// 23: Grpc.CommonService.Enabled: Reference of CommonService.
// 24: Grpc.Interceptors.LoggingZap.Enabled: Enable zap logger interceptor.
// 25: Grpc.Interceptors.MetricsProm.Enabled: Enable prometheus metrics interceptor.
// 26: Grpc.Interceptors.Auth.Enabled: Enable basic auth interceptor.
// 27: Grpc.Interceptors.Auth.Basic: Basic auth credentials as scheme of <user:pass>.
// 28: Grpc.Interceptors.Auth.Bearer: Bearer auth tokens.
// 29: Grpc.Interceptors.Auth.API: API key.
// 30: Grpc.Logger.ZapLogger.Ref: Zap logger reference, see rkentry.ZapLoggerEntry for details.
// 31: Grpc.Logger.EventLogger.Ref: Event logger reference, see rkentry.EventLoggerEntry for details.
type BootConfigGrpc struct {
	Grpc []struct {
		Name        string `yaml:"name" json:"name"`
		Description string `yaml:"description" json:"description"`
		Port        uint64 `yaml:"port" json:"port"`
		Reflection  bool   `yaml:"reflection" json:"reflection"`
		Cert        struct {
			Ref string `yaml:"ref" json:"ref"`
		} `yaml:"cert" json:"cert"`
		GW            BootConfigGw            `yaml:"gw" json:"gw"`
		CommonService BootConfigCommonService `yaml:"commonService" json:"commonService"`
		Interceptors  struct {
			LoggingZap struct {
				Enabled bool `yaml:"enabled" json:"enabled"`
			} `yaml:"loggingZap" json:"loggingZap"`
			MetricsProm struct {
				Enabled bool `yaml:"enabled" json:"enabled"`
			} `yaml:"metricsProm" json:"metricsProm"`
			Auth struct {
				Enabled bool     `yaml:"enabled" json:"enabled"`
				Basic   []string `yaml:"basic" json:"basic"`
				Bearer  []string `yaml:"bearer" json:"bearer"`
				API     []string `yaml:"api" json:"api"`
			} `yaml:"auth" json:"auth"`
			Meta struct {
				Enabled bool   `yaml:"enabled" json:"enabled"`
				Prefix  string `yaml:"prefix" json:"prefix"`
			} `yaml:"meta" json:"meta"`
			TracingTelemetry struct {
				Enabled  bool `yaml:"enabled" json:"enabled"`
				Exporter struct {
					File struct {
						Enabled    bool   `yaml:"enabled" json:"enabled"`
						OutputPath string `yaml:"outputPath" json:"outputPath"`
					} `yaml:"file" json:"file"`
					Jaeger struct {
						Enabled           bool   `yaml:"enabled" json:"enabled"`
						CollectorEndpoint string `yaml:"collectorEndpoint" json:"collectorEndpoint"`
						CollectorUsername string `yaml:"collectorUsername" json:"collectorUsername"`
						CollectorPassword string `yaml:"collectorPassword" json:"collectorPassword"`
					} `yaml:"jaeger" json:"jaeger"`
				} `yaml:"exporter" json:"exporter"`
			} `tracingTelemetry`
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
// 1: ZapLoggerEntry: See rkentry.ZapLoggerEntry for details.
// 2: EventLoggerEntry: See rkentry.EventLoggerEntry for details.
// 3: GwEntry: See GwEntry for details.
// 4: CommonServiceEntry: See CommonServiceEntry for details.
// 5: CertEntry: See CertEntry for details.
// 6: Server: http.Server created while bootstrapping.
// 7: Port: http/https port server listen to.
// 8: ServerOpts: Server options for grpc.
// 9: UnaryInterceptors: Interceptors user enabled from YAML config.
// 10: StreamInterceptors: Interceptors user enabled from YAML config.
// 11: RegFuncs: Grpc registration functions.
// 12: Listener: Listener of grpc.
// 13: EnableReflection: Enable grpc serve reflection.
type GrpcEntry struct {
	EntryName          string                         `json:"entryName" yaml:"entryName"`
	EntryType          string                         `json:"entryType" yaml:"entryType"`
	EntryDescription   string                         `json:"entryDescription" yaml:"entryDescription"`
	ZapLoggerEntry     *rkentry.ZapLoggerEntry        `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	EventLoggerEntry   *rkentry.EventLoggerEntry      `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	GwEntry            *GwEntry                       `json:"gwEntry" yaml:"gwEntry"`
	CommonServiceEntry *CommonServiceEntry            `json:"commonServiceEntry" yaml:"commonServiceEntry"`
	CertEntry          *rkentry.CertEntry             `json:"certEntry" yaml:"certEntry"`
	Server             *grpc.Server                   `json:"-" yaml:"-"`
	Port               uint64                         `json:"port" yaml:"port"`
	ServerOpts         []grpc.ServerOption            `json:"-" yaml:"-"`
	UnaryInterceptors  []grpc.UnaryServerInterceptor  `json:"-" yaml:"-"`
	StreamInterceptors []grpc.StreamServerInterceptor `json:"-" yaml:"-"`
	RegFuncs           []GrpcRegFunc                  `json:"-" yaml:"-"`
	Listener           net.Listener                   `json:"-" yaml:"-"`
	EnableReflection   bool                           `json:"enableReflection" yaml:"enableRefelction"`
}

// Grpc registration func.
type GrpcRegFunc func(server *grpc.Server)

// GrpcEntry option.
type GrpcEntryOption func(*GrpcEntry)

// Register grpc entries with provided config file (Must YAML file).
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

		zapLoggerEntry := rkentry.GlobalAppCtx.GetZapLoggerEntry(element.Logger.ZapLogger.Ref)
		if zapLoggerEntry == nil {
			zapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
		}

		eventLoggerEntry := rkentry.GlobalAppCtx.GetEventLoggerEntry(element.Logger.EventLogger.Ref)
		if eventLoggerEntry == nil {
			eventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
		}

		// Did we enabled gateway?
		var gw *GwEntry
		var commonService *CommonServiceEntry
		var promRegistry *prometheus.Registry
		if element.GW.Enabled {
			// Did we enable common service?
			if element.CommonService.Enabled {
				commonService = NewCommonServiceEntry(
					WithNameCommonService(element.Name),
					WithEventLoggerEntryCommonService(eventLoggerEntry),
					WithZapLoggerEntryCommonService(zapLoggerEntry))
			}

			dialOptions := make([]grpc.DialOption, 0)
			// Did we enabled swagger?
			var sw *SwEntry
			if element.GW.SW.Enabled {
				// init swagger custom headers from config
				headers := make(map[string]string, 0)
				for i := range element.GW.SW.Headers {
					header := element.GW.SW.Headers[i]
					tokens := strings.Split(header, ":")
					if len(tokens) == 2 {
						headers[tokens[0]] = tokens[1]
					}
				}

				sw = NewSwEntry(
					WithNameSw(element.Name),
					WithPortSw(element.GW.Port),
					WithPathSw(element.GW.SW.Path),
					WithJsonPathSw(element.GW.SW.JsonPath),
					WithHeadersSw(headers),
					WithZapLoggerEntrySw(zapLoggerEntry),
					WithEventLoggerEntrySw(eventLoggerEntry),
					WithEnableCommonServiceSw(element.CommonService.Enabled))
			}

			// Did we enable tv?
			var tv *TvEntry
			if element.GW.TV.Enabled {
				tv = NewTvEntry(
					WithNameTv(element.Name),
					WithEventLoggerEntryTv(eventLoggerEntry),
					WithZapLoggerEntryTv(zapLoggerEntry))
			}

			// Did we enable prom?
			var prom *PromEntry
			if element.GW.Prom.Enabled {
				var pusher *rkprom.PushGatewayPusher

				if element.GW.Prom.Pusher.Enabled {
					var certStore *rkentry.CertStore

					if certEntry := rkentry.GlobalAppCtx.GetCertEntry(element.GW.Prom.Pusher.Cert.Ref); certEntry != nil {
						certStore = certEntry.Store
					}

					pusher, _ = rkprom.NewPushGatewayPusher(
						rkprom.WithIntervalMSPusher(time.Duration(element.GW.Prom.Pusher.IntervalMs)*time.Millisecond),
						rkprom.WithRemoteAddressPusher(element.GW.Prom.Pusher.RemoteAddress),
						rkprom.WithJobNamePusher(element.GW.Prom.Pusher.JobName),
						rkprom.WithBasicAuthPusher(element.GW.Prom.Pusher.BasicAuth),
						rkprom.WithZapLoggerEntryPusher(zapLoggerEntry),
						rkprom.WithEventLoggerEntryPusher(eventLoggerEntry),
						rkprom.WithCertStorePusher(certStore))
				}

				promRegistry = prometheus.NewRegistry()
				promRegistry.Register(prometheus.NewGoCollector())
				prom = NewPromEntry(
					WithNameProm(element.Name),
					WithPortProm(element.GW.Port),
					WithPathProm(element.GW.Prom.Path),
					WithZapLoggerEntryProm(zapLoggerEntry),
					WithEventLoggerEntryProm(eventLoggerEntry),
					WithPromRegistryProm(promRegistry),
					WithPusherProm(pusher))
			}

			// Did we enable RK style GW server mux option?
			serverMuxOpts := make([]gwruntime.ServeMuxOption, 0)

			if element.GW.RkServerOption {
				serverMuxOpts = append(serverMuxOpts, RkGwServerMuxOptions...)
			}

			gw = NewGwEntry(
				WithNameGw(element.Name),
				WithZapLoggerEntryGw(zapLoggerEntry),
				WithServerMuxOptionsGw(serverMuxOpts...),
				WithEventLoggerEntryGw(eventLoggerEntry),
				WithGrpcDialOptionsGw(dialOptions...),
				WithGwMappingFilePathsGw(element.GW.GwMappingFilePaths...),
				WithHttpPortGw(element.GW.Port),
				WithGrpcPortGw(element.Port),
				WithCertEntryGw(rkentry.GlobalAppCtx.GetCertEntry(element.GW.Cert.Ref)),
				WithSwEntryGw(sw),
				WithTvEntryGw(tv),
				WithPromEntryGw(prom),
				WithCommonServiceEntryGw(commonService))
		}

		entry := RegisterGrpcEntry(
			WithNameGrpc(element.Name),
			WithDescriptionGrpc(element.Description),
			WithZapLoggerEntryGrpc(zapLoggerEntry),
			WithEventLoggerEntryGrpc(eventLoggerEntry),
			WithPortGrpc(element.Port),
			WithGwEntryGrpc(gw),
			WithCommonServiceEntryGrpc(commonService),
			WithEnableReflectionGrpc(element.Reflection),
			WithCertEntryGrpc(rkentry.GlobalAppCtx.GetCertEntry(element.Cert.Ref)))

		// did we enabled logging interceptor?
		if element.Interceptors.LoggingZap.Enabled {
			opts := make([]rkgrpclog.Option, 0)
			opts = append(opts,
				rkgrpclog.WithEventLoggerEntry(eventLoggerEntry),
				rkgrpclog.WithZapLoggerEntry(zapLoggerEntry),
				rkgrpclog.WithEntryNameAndType(element.Name, GrpcEntryType))

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

		// Did we enabled tracing interceptor?
		if element.Interceptors.TracingTelemetry.Enabled {
			var exporter trace.SpanExporter

			if element.Interceptors.TracingTelemetry.Exporter.File.Enabled {
				exporter = rkgrpctrace.CreateFileExporter(element.Interceptors.TracingTelemetry.Exporter.File.OutputPath)
			}

			if element.Interceptors.TracingTelemetry.Exporter.Jaeger.Enabled {
				exporter = rkgrpctrace.CreateJaegerExporter(
					element.Interceptors.TracingTelemetry.Exporter.Jaeger.CollectorEndpoint,
					element.Interceptors.TracingTelemetry.Exporter.Jaeger.CollectorUsername,
					element.Interceptors.TracingTelemetry.Exporter.Jaeger.CollectorPassword)
			}

			opts := []rkgrpctrace.Option{
				rkgrpctrace.WithEntryNameAndType(element.Name, GrpcEntryType),
				rkgrpctrace.WithExporter(exporter),
			}

			entry.AddUnaryInterceptors(rkgrpctrace.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpctrace.StreamServerInterceptor(opts...))

			rkentry.GlobalAppCtx.AddShutdownHook("tracing exporter", rkgrpctrace.ShutdownExporters)
		}

		// Did we enabled meta interceptor?
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
				rkgrpcauth.WithBearerAuth(element.Interceptors.Auth.Bearer...),
				rkgrpcauth.WithApiKeyAuth(element.Interceptors.Auth.API...))

			entry.AddUnaryInterceptors(rkgrpcauth.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rkgrpcauth.StreamServerInterceptor(opts...))
		}

		res[element.Name] = entry
	}
	return res
}

// Provide name.
func WithNameGrpc(name string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EntryName = name
	}
}

// Provide description.
func WithDescriptionGrpc(description string) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EntryDescription = description
	}
}

// Provide rkentry.ZapLoggerEntry
func WithZapLoggerEntryGrpc(logger *rkentry.ZapLoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ZapLoggerEntry = logger
	}
}

// Provide rkentry.EventLoggerEntry
func WithEventLoggerEntryGrpc(logger *rkentry.EventLoggerEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EventLoggerEntry = logger
	}
}

// Provide port.
func WithPortGrpc(port uint64) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.Port = port
	}
}

// Provide grpc.ServerOption.
func WithServerOptionsGrpc(opts ...grpc.ServerOption) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.ServerOpts = append(entry.ServerOpts, opts...)
	}
}

// Provide grpc.UnaryServerInterceptor.
func WithUnaryInterceptorsGrpc(opts ...grpc.UnaryServerInterceptor) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.UnaryInterceptors = append(entry.UnaryInterceptors, opts...)
	}
}

// Provide grpc.StreamServerInterceptor.
func WithStreamInterceptorsGrpc(opts ...grpc.StreamServerInterceptor) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.StreamInterceptors = append(entry.StreamInterceptors, opts...)
	}
}

// Provide GrpcRegFunc.
func WithGrpcRegFuncsGrpc(funcs ...GrpcRegFunc) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.RegFuncs = append(entry.RegFuncs, funcs...)
	}
}

// Provide GwEntry.
func WithGwEntryGrpc(gw *GwEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.GwEntry = gw
	}
}

// Provide rkentry.CertEntry.
func WithCertEntryGrpc(certEntry *rkentry.CertEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.CertEntry = certEntry
	}
}

// Provide CommonServiceEntry.
func WithCommonServiceEntryGrpc(commonService *CommonServiceEntry) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.CommonServiceEntry = commonService
	}
}

// Provice EnableReflection.
func WithEnableReflectionGrpc(enabled bool) GrpcEntryOption {
	return func(entry *GrpcEntry) {
		entry.EnableReflection = enabled
	}
}

// Register GrpcEntry with options.
func RegisterGrpcEntry(opts ...GrpcEntryOption) *GrpcEntry {
	entry := &GrpcEntry{
		EntryType:          GrpcEntryType,
		EntryDescription:   GrpcEntryDescription,
		ZapLoggerEntry:     rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry:   rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		ServerOpts:         make([]grpc.ServerOption, 0),
		UnaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		StreamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		RegFuncs:           make([]GrpcRegFunc, 0),
		Port:               1949,
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
	//    +-------+
	//    | panic |
	//    +-------+
	// Append panic interceptor at the end
	entry.UnaryInterceptors = append(entry.UnaryInterceptors, rkgrpcpanic.UnaryServerInterceptor())
	entry.StreamInterceptors = append(entry.StreamInterceptors, rkgrpcpanic.StreamServerInterceptor())

	if entry.ZapLoggerEntry == nil {
		entry.ZapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
	}

	if entry.EventLoggerEntry == nil {
		entry.EventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = "GrpcServer-" + strconv.FormatUint(entry.Port, 10)
	}

	if entry.CommonServiceEntry != nil {
		entry.RegFuncs = append(entry.RegFuncs, entry.CommonServiceEntry.RegFuncGrpc)
	}

	rkentry.GlobalAppCtx.AddEntry(entry)

	return entry
}

// Add grpc server options.
func (entry *GrpcEntry) AddServerOptions(opts ...grpc.ServerOption) {
	entry.ServerOpts = append(entry.ServerOpts, opts...)
}

// Add unary interceptor.
func (entry *GrpcEntry) AddUnaryInterceptors(inter ...grpc.UnaryServerInterceptor) {
	entry.UnaryInterceptors = append(entry.UnaryInterceptors, inter...)
}

// Add stream interceptor.
func (entry *GrpcEntry) AddStreamInterceptors(inter ...grpc.StreamServerInterceptor) {
	entry.StreamInterceptors = append(entry.StreamInterceptors, inter...)
}

// Add grpc registration func.
func (entry *GrpcEntry) AddGrpcRegFuncs(funcs ...GrpcRegFunc) {
	entry.RegFuncs = append(entry.RegFuncs, funcs...)
}

// Add gateway registration func.
func (entry *GrpcEntry) AddGwRegFuncs(funcs ...GwRegFunc) {
	if entry.GwEntry != nil {
		entry.GwEntry.addRegFuncsGw(funcs...)
	}
}

// Get entry name.
func (entry *GrpcEntry) GetName() string {
	return entry.EntryName
}

// Get entry type.
func (entry *GrpcEntry) GetType() string {
	return entry.EntryType
}

// Stringfy entry.
func (entry *GrpcEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// Get description of entry.
func (entry *GrpcEntry) GetDescription() string {
	return entry.EntryDescription
}

// Bootstrap GrpcEntry.
func (entry *GrpcEntry) Bootstrap(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"bootstrap",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	ctx = context.WithValue(context.Background(), bootstrapEventIdKey, event.GetEventId())
	logger := entry.ZapLoggerEntry.GetLogger().With(zap.String("eventId", event.GetEventId()))

	entry.logBasicInfo(event)

	// Common service enabled?
	if entry.IsCommonServiceEnabled() {
		entry.CommonServiceEntry.Bootstrap(ctx)
	}

	// Gateway enabled?
	// Start gateway first since we do not want to block goroutine here
	if entry.IsGwEnabled() {
		entry.GwEntry.Bootstrap(ctx)
	}

	listener, err := net.Listen("tcp4", ":"+strconv.FormatUint(entry.Port, 10))
	if err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		rkcommon.ShutdownWithError(err)
	}

	entry.Listener = listener

	if entry.IsTlsEnabled() {
		if cert, err := tls.X509KeyPair(entry.CertEntry.Store.ServerCert, entry.CertEntry.Store.ServerKey); err != nil {
			rkcommon.ShutdownWithError(err)
		} else {
			tls := credentials.NewServerTLSFromCert(&cert)
			entry.ServerOpts = append(entry.ServerOpts, grpc.Creds(tls))
		}
	}

	// make unary and stream interceptors into server opts
	entry.ServerOpts = append(entry.ServerOpts,
		grpc.ChainUnaryInterceptor(entry.UnaryInterceptors...),
		grpc.ChainStreamInterceptor(entry.StreamInterceptors...))

	// create grpc server
	entry.Server = grpc.NewServer(entry.ServerOpts...)
	for _, regFunc := range entry.RegFuncs {
		regFunc(entry.Server)
	}

	// enable reflection
	if entry.EnableReflection {
		reflection.Register(entry.Server)
	}

	logger.Info("Bootstrapping grpcEntry.", event.ListPayloads()...)
	go func(*GrpcEntry) {
		// start grpc server
		if err := entry.Server.Serve(listener); err != nil {
			event.AddErr(err)
			logger.Error("Error occurs while serving grpc-server.", event.ListPayloads()...)
			rkcommon.ShutdownWithError(err)
		}
	}(entry)

	entry.EventLoggerEntry.GetEventHelper().Finish(event)
}

// Interrupt GrpcEntry.
func (entry *GrpcEntry) Interrupt(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"interrupt",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	ctx = context.WithValue(context.Background(), bootstrapEventIdKey, event.GetEventId())
	logger := entry.ZapLoggerEntry.GetLogger().With(zap.String("eventId", event.GetEventId()))

	entry.logBasicInfo(event)

	if entry.IsCommonServiceEnabled() {
		entry.CommonServiceEntry.Interrupt(ctx)
	}

	if entry.IsGwEnabled() {
		entry.GwEntry.Interrupt(ctx)
	}

	logger.Info("Interrupting grpcEntry.", event.ListPayloads()...)

	if entry.Server != nil {
		entry.Server.GracefulStop()
	}

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)
}

// Marshal entry.
func (entry *GrpcEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":          entry.EntryName,
		"entryType":          entry.EntryType,
		"entryDescription":   entry.EntryDescription,
		"eventLoggerEntry":   entry.EventLoggerEntry.GetName(),
		"zapLoggerEntry":     entry.ZapLoggerEntry.GetName(),
		"port":               entry.Port,
		"gwEntry":            entry.GwEntry,
		"commonServiceEntry": entry.CommonServiceEntry,
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

	regFuncsStr := make([]string, 0)
	m["regFuncs"] = &serverOptsStr

	for i := range entry.RegFuncs {
		element := entry.RegFuncs[i]
		regFuncsStr = append(regFuncsStr,
			runtime.FuncForPC(reflect.ValueOf(element).Pointer()).Name())
	}

	return json.Marshal(&m)
}

// Not supported.
func (entry *GrpcEntry) UnmarshalJSON([]byte) error {
	return nil
}

// Is TLS enabled?
func (entry *GrpcEntry) IsTlsEnabled() bool {
	return entry.CertEntry != nil && entry.CertEntry.Store != nil
}

// Is grpc gateway enabled?
func (entry *GrpcEntry) IsGwEnabled() bool {
	return entry.GwEntry != nil
}

// Is common service enabled?
func (entry *GrpcEntry) IsCommonServiceEnabled() bool {
	return entry.CommonServiceEntry != nil
}

// Add basic fields into event.
func (entry *GrpcEntry) logBasicInfo(event rkquery.Event) {
	event.AddPayloads(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
		zap.Uint64("grpcPort", entry.Port),
		zap.Bool("commonServiceEnabled", entry.IsCommonServiceEnabled()),
		zap.Bool("tlsEnabled", entry.IsTlsEnabled()),
		zap.Bool("gwEnabled", entry.IsGwEnabled()),
		zap.Bool("reflectionEnabled", entry.EnableReflection))

	if entry.IsGwEnabled() {
		event.AddPayloads(
			zap.Bool("swEnabled", entry.GwEntry.IsSwEnabled()),
			zap.Bool("tvEnabled", entry.GwEntry.IsTvEnabled()),
			zap.Bool("promEnabled", entry.GwEntry.IsPromEnabled()),
			zap.Bool("gwClientTlsEnabled", entry.GwEntry.IsClientTlsEnabled()),
			zap.Bool("gwServerTlsEnabled", entry.GwEntry.IsServerTlsEnabled()))

		if entry.GwEntry.IsSwEnabled() {
			event.AddPayloads(
				zap.String("swPath", entry.GwEntry.SwEntry.Path),
				zap.Any("headers", entry.GwEntry.SwEntry.Headers))
		}

		if entry.GwEntry.IsTvEnabled() {
			event.AddPayloads(
				zap.String("tvPath", "/rk/v1/tv"))
		}
	}
}

// Get GinEntry from rkentry.GlobalAppCtx.
func GetGrpcEntry(name string) *GrpcEntry {
	entryRaw := rkentry.GlobalAppCtx.GetEntry(name)
	if entryRaw == nil {
		return nil
	}

	entry, _ := entryRaw.(*GrpcEntry)
	return entry
}