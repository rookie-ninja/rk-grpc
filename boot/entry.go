// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc

import (
	"crypto/tls"
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/rookie-ninja/rk-common/context"
	rk_entry "github.com/rookie-ninja/rk-common/entry"
	"github.com/rookie-ninja/rk-grpc/boot/api/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"strconv"
	"strings"
	"time"
)

func init() {
	rk_ctx.RegisterEntryInitializer(NewGRpcEntries)
}

type bootConfig struct {
	GRpc []struct {
		Name                string `yaml:"name"`
		Port                uint64 `yaml:"port"`
		EnableCommonService bool   `yaml:"enableCommonService"`
		TLS                 struct {
			Enabled bool `yaml:"enabled"`
			User    struct {
				Enabled  bool   `yaml:"enabled"`
				CertFile string `yaml:"certFile"`
				KeyFile  string `yaml:"keyFile"`
			} `yaml:"user"`
			Auto struct {
				Enabled    bool   `yaml:"enabled"`
				CertOutput string `yaml:"certOutput"`
			} `yaml:"auto"`
		} `yaml:"tls"`
		GW struct {
			Enabled  bool   `yaml:"enabled"`
			Port     uint64 `yaml:"port"`
			EnableTV bool   `yaml:"enableTV"`
			SW       struct {
				Enabled  bool     `yaml:"enabled"`
				Path     string   `yaml:"path"`
				JSONPath string   `yaml:"jsonPath"`
				Headers  []string `yaml:"headers"`
			} `yaml:"sw"`
		} `yaml:"gw"`
		LoggingInterceptor struct {
			Enabled              bool `yaml:"enabled"`
			EnableLogging        bool `yaml:"enableLogging"`
			EnableMetrics        bool `yaml:"enableMetrics"`
			EnablePayloadLogging bool `yaml:"enablePayloadLogging"`
		} `yaml:"loggingInterceptor"`
	} `yaml:"grpc"`
}

type GRpcEntry struct {
	entryType           string
	logger              *zap.Logger
	name                string
	port                uint64
	enableCommonService bool
	serverOpts          []grpc.ServerOption
	unaryInterceptors   []grpc.UnaryServerInterceptor
	streamInterceptors  []grpc.StreamServerInterceptor
	regFuncs            []RegFuncGRpc
	gw                  *gwEntry
	tls                 *tlsEntry
	server              *grpc.Server
	listener            net.Listener
}

type RegFuncGRpc func(server *grpc.Server)

type GRpcEntryOption func(*GRpcEntry)

func NewGRpcEntries(path string, factory *rk_query.EventFactory, logger *zap.Logger) map[string]rk_entry.Entry {
	bytes := readFile(path)
	config := &bootConfig{}
	if err := yaml.Unmarshal(bytes, config); err != nil {
		return nil
	}

	return getGRpcServerEntries(config, factory, logger)
}

func getGRpcServerEntries(config *bootConfig, factory *rk_query.EventFactory, logger *zap.Logger) map[string]rk_entry.Entry {
	res := make(map[string]rk_entry.Entry)

	for i := range config.GRpc {
		element := config.GRpc[i]
		name := element.Name

		// did we enabled tls?
		var tls *tlsEntry
		if element.TLS.Enabled {
			if element.TLS.User.Enabled {
				tls = newTlsEntry(
					withCertFilePath(element.TLS.User.CertFile),
					withKeyFilePath(element.TLS.User.KeyFile))
			} else if element.TLS.Auto.Enabled {
				tls = newTlsEntry(
					withGenerateCert(element.TLS.Auto.Enabled),
					withGeneratePath(element.TLS.Auto.CertOutput))
			}
		}

		// did we enabled gateway?
		var gw *gwEntry
		if element.GW.Enabled {
			opts := make([]grpc.DialOption, 0)

			// did we enabled swagger?
			var sw *swEntry
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

				sw = newSWEntry(
					withSWPath(element.GW.SW.Path),
					withSWJsonPath(element.GW.SW.JSONPath),
					withHeaders(headers))
			}

			gw = newGRpcGWEntry(
				withHttpPortGW(element.GW.Port),
				withGRpcPortGW(element.Port),
				withDialOptionsGW(opts...),
				withSWEntryGW(sw),
				withTlsEntryGW(tls),
				withLoggerGW(logger),
				withEnableCommonServiceGW(element.EnableCommonService),
				withEnableTV(element.GW.EnableTV))
		}

		entry := NewGRpcEntry(
			WithName(name),
			WithPort(element.Port),
			WithGWEntry(gw),
			WithRegFuncs(registerRkCommonService),
			WithCommonService(element.EnableCommonService),
			WithTlsEntry(tls),
			WithLogger(logger))

		// did we enabled logging interceptor?
		if element.LoggingInterceptor.Enabled {
			opts := make([]rk_grpc_log.Option, 0)
			opts = append(opts,
				rk_grpc_log.WithEnableLogging(element.LoggingInterceptor.EnableLogging),
				rk_grpc_log.WithEnableMetrics(element.LoggingInterceptor.EnableMetrics),
				rk_grpc_log.WithEnablePayloadLogging(element.LoggingInterceptor.EnablePayloadLogging),
				rk_grpc_log.WithEventFactory(factory),
				rk_grpc_log.WithLogger(logger))

			entry.AddUnaryInterceptors(rk_grpc_log.UnaryServerInterceptor(opts...))
			entry.AddStreamInterceptors(rk_grpc_log.StreamServerInterceptor(opts...))
		}

		res[name] = entry
	}
	return res
}

func WithRegFuncs(funcs ...RegFuncGRpc) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.regFuncs = append(entry.regFuncs, funcs...)
	}
}

func WithName(name string) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.name = name
	}
}

func WithGWEntry(gw *gwEntry) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.gw = gw
	}
}

func WithTlsEntry(tls *tlsEntry) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.tls = tls
	}
}

func WithLogger(log *zap.Logger) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.logger = log
	}
}

func WithPort(port uint64) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.port = port
	}
}

func WithCommonService(enable bool) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.enableCommonService = enable
	}
}

func WithServerOptions(opts ...grpc.ServerOption) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.serverOpts = append(entry.serverOpts, opts...)
	}
}

func WithUnaryInterceptors(opts ...grpc.UnaryServerInterceptor) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.unaryInterceptors = append(entry.unaryInterceptors, opts...)
	}
}

func WithStreamInterceptors(opts ...grpc.StreamServerInterceptor) GRpcEntryOption {
	return func(entry *GRpcEntry) {
		entry.streamInterceptors = append(entry.streamInterceptors, opts...)
	}
}

func NewGRpcEntry(opts ...GRpcEntryOption) *GRpcEntry {
	entry := &GRpcEntry{
		entryType:          "grpc",
		logger:             zap.NewNop(),
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
	}

	for i := range opts {
		opts[i](entry)
	}

	if len(entry.name) < 1 {
		entry.name = "gRpc-server-" + strconv.FormatUint(entry.port, 10)
	}

	if entry.serverOpts == nil {
		entry.serverOpts = make([]grpc.ServerOption, 0)
	}

	if entry.regFuncs == nil {
		entry.regFuncs = make([]RegFuncGRpc, 0)
	}

	rk_ctx.GlobalAppCtx.AddEntry(entry.GetName(), entry)

	return entry
}

func (entry *GRpcEntry) AddServerOptions(opts ...grpc.ServerOption) {
	entry.serverOpts = append(entry.serverOpts, opts...)
}

func (entry *GRpcEntry) AddUnaryInterceptors(inter ...grpc.UnaryServerInterceptor) {
	entry.unaryInterceptors = append(entry.unaryInterceptors, inter...)
}

func (entry *GRpcEntry) AddStreamInterceptors(inter ...grpc.StreamServerInterceptor) {
	entry.streamInterceptors = append(entry.streamInterceptors, inter...)
}

func (entry *GRpcEntry) AddGRpcRegFuncs(funcs ...RegFuncGRpc) {
	entry.regFuncs = append(entry.regFuncs, funcs...)
}

func (entry *GRpcEntry) AddGWRegFuncs(funcs ...regFuncGW) {
	if entry.gw != nil {
		entry.gw.addRegFuncsGW(funcs...)
	}
}

func (entry *GRpcEntry) GetPort() uint64 {
	return entry.port
}

func (entry *GRpcEntry) GetName() string {
	return entry.name
}

func (entry *GRpcEntry) GetType() string {
	return entry.entryType
}

func (entry *GRpcEntry) IsTlsEnabled() bool {
	return entry.tls != nil
}

func (entry *GRpcEntry) IsGWEnabled() bool {
	return entry.gw != nil
}

func (entry *GRpcEntry) String() string {
	m := map[string]string{
		"name":                entry.GetName(),
		"type":                entry.GetType(),
		"grpc_port":           strconv.FormatUint(entry.GetPort(), 10),
		"unary_interceptors":  strconv.Itoa(len(entry.unaryInterceptors)),
		"stream_interceptors": strconv.Itoa(len(entry.streamInterceptors)),
		"tls":                 strconv.FormatBool(entry.IsTlsEnabled()),
		"gw":                  strconv.FormatBool(entry.IsGWEnabled()),
	}

	if entry.IsGWEnabled() {
		m["gw_port"] = strconv.FormatUint(entry.gw.GetHttpPort(), 10)
		m["sw"] = strconv.FormatBool(entry.GetGWEntry().isSWEnabled())

		if entry.GetGWEntry().isSWEnabled() {
			m["sw_path"] = entry.GetGWEntry().getSWEntry().GetPath()
		}
	}

	bytes, _ := json.Marshal(m)

	return string(bytes)
}

func (entry *GRpcEntry) GetServer() *grpc.Server {
	return entry.server
}

func (entry *GRpcEntry) GetListener() net.Listener {
	return entry.listener
}

func (entry *GRpcEntry) GetGWEntry() *gwEntry {
	return entry.gw
}

func (entry *GRpcEntry) Shutdown(event rk_query.Event) {
	if entry.server != nil {
		fields := []zap.Field{
			zap.Uint64("grpc_port", entry.GetPort()),
			zap.String("name", entry.name),
		}

		if entry.tls != nil {
			fields = append(fields, zap.Bool("tls", true))
		}

		if entry.gw != nil {
			entry.gw.Shutdown(event)
		}

		event.AddFields(fields...)

		entry.logger.Info("stopping grpc-server", fields...)
		entry.server.GracefulStop()
	}
}

func (entry *GRpcEntry) Bootstrap(event rk_query.Event) {
	fields := []zap.Field{
		zap.Uint64("grpc_port", entry.port),
		zap.String("grpc_name", entry.name),
	}

	// gateway enabled?
	// start gateway first since we do not want to block goroutine here
	if entry.gw != nil {
		entry.gw.Bootstrap(event)
	}

	listener, err := net.Listen("tcp4", ":"+strconv.FormatUint(entry.port, 10))
	if err != nil {
		shutdownWithError(err)
	}

	entry.listener = listener

	// make unary server opts
	entry.unaryInterceptors = append(entry.unaryInterceptors,
		rk_grpc_panic.UnaryServerInterceptor(rk_grpc_panic.PanicToZap))
	entry.serverOpts = append(entry.serverOpts,
		grpc.ChainUnaryInterceptor(entry.unaryInterceptors...))

	// make stream server opts
	// we only need to add panic handler once
	// since we use a global variable storing handlers which is accessible by
	// unary and stream interceptor
	entry.serverOpts = append(entry.serverOpts,
		grpc.ChainStreamInterceptor(entry.streamInterceptors...))

	// tls enabled?
	if entry.tls != nil {
		cert, err := tls.LoadX509KeyPair(entry.tls.getCertFilePath(), entry.tls.getKeyFilePath())
		if err != nil {
			shutdownWithError(err)
		}
		entry.serverOpts = append(entry.serverOpts, grpc.Creds(credentials.NewServerTLSFromCert(&cert)))
		fields = append(fields, zap.Bool("grpc_tls", true))
	}

	// creat grpc server
	entry.server = grpc.NewServer(entry.serverOpts...)
	for _, regFunc := range entry.regFuncs {
		regFunc(entry.server)
	}

	event.AddFields(fields...)
	// start grpc server
	entry.logger.Info("starting grpc-server", fields...)
	go func(*GRpcEntry) {
		if err := entry.server.Serve(listener); err != nil {
			fields = append(fields, zap.Error(err))
			entry.logger.Error("err while serving grpc-listener", fields...)
			shutdownWithError(err)
		}
	}(entry)
}

func (entry *GRpcEntry) Wait(draining time.Duration) {
	sig := <-rk_ctx.GlobalAppCtx.GetShutdownSig()

	helper := rk_query.NewEventHelper(rk_ctx.GlobalAppCtx.GetEventFactory())
	event := helper.Start("rk_app_stop")

	rk_ctx.GlobalAppCtx.GetDefaultLogger().Info("draining", zap.Duration("draining_duration", draining))
	time.Sleep(draining)

	event.AddFields(
		zap.Duration("app_lifetime_nano", time.Since(rk_ctx.GlobalAppCtx.GetStartTime())),
		zap.Time("app_start_time", rk_ctx.GlobalAppCtx.GetStartTime()))

	event.AddPair("signal", sig.String())

	entry.Shutdown(event)

	helper.Finish(event)
}

// Register common service
func registerRkCommonService(server *grpc.Server) {
	rk_grpc_common_v1.RegisterRkCommonServiceServer(server, NewCommonServiceGRpc())
}
