// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc

import (
	"context"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rookie-ninja/rk-grpc/boot/api/v1"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
)

type gwEntry struct {
	logger              *zap.Logger
	httpPort            uint64
	gRpcPort            uint64
	enableCommonService bool
	enableTV			bool
	tls                 *tlsEntry
	sw                  *swEntry
	regFuncs            []regFuncGW
	dialOpts            []grpc.DialOption
	muxOpts             []runtime.ServeMuxOption
	server              *http.Server
	gwMux               *runtime.ServeMux
	mux                 *http.ServeMux
}

type regFuncGW func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

type gRpcGWOption func(*gwEntry)

func withHttpPortGW(port uint64) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.httpPort = port
	}
}

func withLoggerGW(log *zap.Logger) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.logger = log
	}
}

func withTlsEntryGW(tls *tlsEntry) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.tls = tls
	}
}

func withEnableCommonServiceGW(enable bool) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.enableCommonService = enable
	}
}

func withEnableTV(enable bool) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.enableTV = enable
	}
}

func withSWEntryGW(sw *swEntry) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.sw = sw
	}
}

func withGRpcPortGW(port uint64) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.gRpcPort = port
	}
}

func withRegFuncsGW(funcs ...regFuncGW) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.regFuncs = append(entry.regFuncs, funcs...)
	}
}

func withDialOptionsGW(opts ...grpc.DialOption) gRpcGWOption {
	return func(entry *gwEntry) {
		entry.dialOpts = append(entry.dialOpts, opts...)
	}
}

func newGRpcGWEntry(opts ...gRpcGWOption) *gwEntry {
	entry := &gwEntry{
		logger: zap.NewNop(),
	}

	for i := range opts {
		opts[i](entry)
	}

	if entry.dialOpts == nil {
		entry.dialOpts = make([]grpc.DialOption, 0)
	}

	if entry.regFuncs == nil {
		entry.regFuncs = make([]regFuncGW, 0)
	}

	if entry.enableCommonService {
		entry.regFuncs = append(entry.regFuncs, rk_grpc_common_v1.RegisterRkCommonServiceHandlerFromEndpoint)
	}

	return entry
}

func (entry *gwEntry) addDialOptions(opts ...grpc.DialOption) {
	entry.dialOpts = append(entry.dialOpts, opts...)
}

func (entry *gwEntry) addRegFuncsGW(funcs ...regFuncGW) {
	entry.regFuncs = append(entry.regFuncs, funcs...)
}

func (entry *gwEntry) isSWEnabled() bool {
	return entry.sw != nil
}

func (entry *gwEntry) getSWEntry() *swEntry {
	return entry.sw
}

func (entry *gwEntry) GetHttpPort() uint64 {
	return entry.httpPort
}

func (entry *gwEntry) GetGRpcPort() uint64 {
	return entry.gRpcPort
}

func (entry *gwEntry) GetServer() *http.Server {
	return entry.server
}

func (entry *gwEntry) Shutdown(event rk_query.Event) {
	fields := []zap.Field{
		zap.Uint64("grpc_gw_port", entry.httpPort),
	}

	if entry.sw != nil {
		fields = append(fields, zap.String("grpc_sw_path", entry.sw.GetPath()))
	}

	event.AddFields(fields...)

	if entry.server != nil {
		entry.logger.Info("stopping grpc-gateway", fields...)
		if err := entry.server.Shutdown(context.Background()); err != nil {
			fields = append(fields, zap.Error(err))
			entry.logger.Warn("error occurs while stopping gRpc-gateway", fields...)
		}
	}
}

func (entry *gwEntry) Bootstrap(event rk_query.Event) {
	// init tls server only if port is not zero
	if entry.tls != nil {
		creds, err := credentials.NewClientTLSFromFile(entry.tls.getCertFilePath(), "")
		if err != nil {
			shutdownWithError(err)
		}

		entry.addDialOptions(grpc.WithTransportCredentials(creds))
	} else {
		entry.addDialOptions(grpc.WithInsecure())
	}

	gRPCEndpoint := "0.0.0.0:" + strconv.FormatUint(entry.gRpcPort, 10)
	// use proto names for return value instead of camel case
	entry.muxOpts = append(entry.muxOpts,
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb {
			MarshalOptions: protojson.MarshalOptions {
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{},
		}),
		runtime.WithOutgoingHeaderMatcher(OutgoingHeaderMatcher))

	entry.gwMux = runtime.NewServeMux(entry.muxOpts...)

	for i := range entry.regFuncs {
		err := entry.regFuncs[i](context.Background(), entry.gwMux, gRPCEndpoint, entry.dialOpts)
		if err != nil {
			fields := []zap.Field{
				zap.Uint64("http_port", entry.httpPort),
				zap.Uint64("gRpc_port", entry.gRpcPort),
				zap.Error(err),
			}
			if entry.tls != nil {
				fields = append(fields, zap.Bool("tls", true))
			}

			entry.logger.Error("registering functions", fields...)
			shutdownWithError(err)
		}
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", entry.gwMux)

	// register tv handler
	if entry.enableTV {
		httpMux.HandleFunc("/v1/rk/tv/", tv)
	}

	// support swagger
	if entry.sw != nil {
		httpMux.HandleFunc(swHandlerPrefix, entry.sw.swJsonFileHandler)
		httpMux.HandleFunc(entry.sw.path, entry.sw.swIndexHandler)
	}

	entry.server = &http.Server{
		Addr:    "0.0.0.0:" + strconv.FormatUint(entry.httpPort, 10),
		Handler: headMethodHandler(httpMux),
	}

	entry.mux = httpMux

	fields := []zap.Field{
		zap.Uint64("grpc_gw_port", entry.httpPort),
	}

	if entry.sw != nil {
		fields = append(fields, zap.String("grpc_sw_path", entry.sw.GetPath()))
	}

	if entry.tls != nil {
		fields = append(fields, zap.Bool("grpc_tls", true))
	}

	event.AddFields(fields...)

	entry.logger.Info("starting grpc-gateway", fields...)
	if entry.tls != nil {
		go func(*gwEntry) {
			if err := entry.server.ListenAndServeTLS(entry.tls.getCertFilePath(), entry.tls.getKeyFilePath()); err != nil && err != http.ErrServerClosed {
				fields = append(fields, zap.Error(err))
				entry.logger.Error("failed to start grpc-gateway", fields...)
				shutdownWithError(err)
			}
		}(entry)
	} else {
		go func(*gwEntry) {
			if err := entry.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fields = append(fields, zap.Error(err))
				entry.logger.Error("failed to start grpc-gateway", fields...)
				shutdownWithError(err)
			}
		}(entry)
	}
}

// Support HEAD request
func headMethodHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			return
		}
		h.ServeHTTP(w, r)
	})
}

func readFile(filePath string) []byte {
	if !path.IsAbs(filePath) {
		wd, err := os.Getwd()

		if err != nil {
			shutdownWithError(err)
		}
		filePath = path.Join(wd, filePath)
	}

	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		shutdownWithError(err)
	}

	return bytes
}

// without prefix
func OutgoingHeaderMatcher(key string) (string, bool) {
	return key, true
}