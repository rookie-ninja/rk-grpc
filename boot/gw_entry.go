// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/boot/api/gen/v1"
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
	"strings"
)

const (
	GwEntryType        = "GwEntry"
	GwEntryNameDefault = "GwDefault"
	GwEntryDescription = "Internal RK entry which implements grpc gateway on top of grpc framework."
)

type gwRule struct {
	Method  string `json:"method" yaml:"method"`
	Pattern string `json:"pattern" yaml:"pattern"`
}

// Bootstrap config of tv.
// 1: Enabled: Enable gateway.
// 2: Port: Http port exposed.
// 3: Enable RK sytle server option?
// 4: Cert.Ref: Reference of rkentry.CertEntry.
// 5: Logger.ZapLogger.Ref: Reference of rkentry.ZapLoggerEntry.
// 6: Logger.EventLogger.Ref: Reference of rkentry.EventLoggerEntry.
// 7: GwMappingFilePaths: Array of file path of gateway file path file.
// 8: Tv: See BootConfigTv for details.
// 9: Sw: See BootConfigSw for details.
// 10: Prom: See BootConfigProm for details.
type BootConfigGw struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Port           uint64 `yaml:"port" json:"port"`
	RkServerOption bool   `yaml:"rkServerOption" json:"rkServerOption"`
	Cert           struct {
		Ref string `yaml:"ref" json:"ref"`
	} `yaml:"cert" json:"cert"`
	GwMappingFilePaths []string       `yaml:"gwMappingFilePaths" json:"gwMappingFilePaths"`
	TV                 BootConfigTv   `yaml:"tv" json:"tv"`
	SW                 BootConfigSw   `yaml:"sw" json:"sw"`
	Prom               BootConfigProm `yaml:"prom" json:"prom"`
}

// GwEntry implements rkentry.Entry interface.
//
// 1: GwMappingFilePaths: The paths of gateway mapping file, either relative or absolute path is acceptable.
// 2: HttpPort: Http port.
// 3: GrpcPort: Grpc port.
// 4: ZapLoggerEntry: See rkentry.ZapLoggerEntry for details.
// 5: EventLoggerEntry: See rkentry.EventLoggerEntry for details.
// 6: CertEntry: See rkentry.CertEntry for details.
// 7: SwEntry: See SwEntry for details.
// 8: TvEntry: See TvEntry for details.
// 9: PromEntry: See PromEntry for details.
// 10: CommonServiceEntry: See CommonServiceEntry for details.
// 11: Server: http.Server created while bootstrapping.
// 12: RegFuncsGw: Registration function for grpc gateway.
// 13: GrpcDialOptions: Grpc dial options.
// 14: ServerMuxOptions: Grpc gateway server options.
// 15: Server: http.Server for grpc gateway.
// 16: GwMux: runtime.ServeMux.
// 17: Mux: http.ServeMux.
type GwEntry struct {
	EntryName          string                    `json:"entryName" yaml:"entryName"`
	EntryType          string                    `json:"entryType" yaml:"entryType"`
	EntryDescription   string                    `json:"entryDescription" yaml:"entryDescription"`
	GwMappingFilePaths []string                  `json:"gwMappingFilePaths" yaml:"gwMappingFilePaths"`
	GwMapping          map[string]*gwRule        `json:"gwMapping" yaml:"gwMapping"`
	HttpPort           uint64                    `json:"httpPort" yaml:"httpPort"`
	GrpcPort           uint64                    `json:"grpcPort" yaml:"grpcPort"`
	ZapLoggerEntry     *rkentry.ZapLoggerEntry   `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	EventLoggerEntry   *rkentry.EventLoggerEntry `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	CertEntry          *rkentry.CertEntry        `json:"certEntry" yaml:"certEntry"`
	GrpcCertEntry      *rkentry.CertEntry        `json:"grpcCertEntry" yaml:"grpcCertEntry"`
	SwEntry            *SwEntry                  `json:"swEntry" yaml:"swEntry"`
	TvEntry            *TvEntry                  `json:"tvEntry" yaml:"tvEntry"`
	PromEntry          *PromEntry                `json:"promEntry" yaml:"promEntry"`
	CommonServiceEntry *CommonServiceEntry       `json:"commonServiceEntry" yaml:"commonServiceEntry"`
	RegFuncsGw         []GwRegFunc               `json:"-" yaml:"-"`
	GrpcDialOptions    []grpc.DialOption         `json:"-" yaml:"-"`
	ServerMuxOptions   []runtime.ServeMuxOption  `json:"-" yaml:"-"`
	Server             *http.Server              `json:"-" yaml:"-"`
	GwMux              *runtime.ServeMux         `json:"-" yaml:"-"`
	Mux                *http.ServeMux            `json:"-" yaml:"-"`
}

// Registration function grpc gateway.
type GwRegFunc func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

// GwEntry option.
type GwOption func(*GwEntry)

// Provide name for gateway.
func WithNameGw(name string) GwOption {
	return func(entry *GwEntry) {
		entry.EntryName = name
	}
}

// Provide gateway mapping configuration file paths.
func WithGwMappingFilePathsGw(paths ...string) GwOption {
	return func(entry *GwEntry) {
		entry.GwMappingFilePaths = append(entry.GwMappingFilePaths, paths...)
	}
}

// Provide rkentry.ZapLoggerEntry.
func WithZapLoggerEntryGw(zapLoggerEntry *rkentry.ZapLoggerEntry) GwOption {
	return func(entry *GwEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// Provide rkentry.EventLoggerEntry.
func WithEventLoggerEntryGw(eventLoggerEntry *rkentry.EventLoggerEntry) GwOption {
	return func(entry *GwEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// Provide http port.
func WithHttpPortGw(port uint64) GwOption {
	return func(entry *GwEntry) {
		entry.HttpPort = port
	}
}

// Provide grpc port.
func WithGrpcPortGw(port uint64) GwOption {
	return func(entry *GwEntry) {
		entry.GrpcPort = port
	}
}

// Provide rkentry.CertEntry.
func WithCertEntryGw(certEntry *rkentry.CertEntry) GwOption {
	return func(entry *GwEntry) {
		entry.CertEntry = certEntry
	}
}

// Provide rkentry.CertEntry.
func WithGrpcCertEntryGw(grpcCertEntry *rkentry.CertEntry) GwOption {
	return func(entry *GwEntry) {
		entry.GrpcCertEntry = grpcCertEntry
	}
}

// Provide SwEntry.
func WithSwEntryGw(sw *SwEntry) GwOption {
	return func(entry *GwEntry) {
		entry.SwEntry = sw
	}
}

// Provide TvEntry.
func WithTvEntryGw(tv *TvEntry) GwOption {
	return func(entry *GwEntry) {
		entry.TvEntry = tv
	}
}

// Provide PromEntry.
func WithPromEntryGw(prom *PromEntry) GwOption {
	return func(entry *GwEntry) {
		entry.PromEntry = prom
	}
}

// Provide CommonServiceEntry.
func WithCommonServiceEntryGw(commonService *CommonServiceEntry) GwOption {
	return func(entry *GwEntry) {
		entry.CommonServiceEntry = commonService
	}
}

// Provide registration function.
func WithRegFuncsGw(funcs ...GwRegFunc) GwOption {
	return func(entry *GwEntry) {
		entry.RegFuncsGw = append(entry.RegFuncsGw, funcs...)
	}
}

// Provide grpc dial options.
func WithGrpcDialOptionsGw(opts ...grpc.DialOption) GwOption {
	return func(entry *GwEntry) {
		entry.GrpcDialOptions = append(entry.GrpcDialOptions, opts...)
	}
}

// Provide gateway server mux options.
func WithServerMuxOptionsGw(opts ...runtime.ServeMuxOption) GwOption {
	return func(entry *GwEntry) {
		entry.ServerMuxOptions = append(entry.ServerMuxOptions, opts...)
	}
}

// Create new gateway entry with options.
func NewGwEntry(opts ...GwOption) *GwEntry {
	entry := &GwEntry{
		EntryName:          GwEntryNameDefault,
		EntryType:          GwEntryType,
		EntryDescription:   GwEntryDescription,
		ZapLoggerEntry:     rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry:   rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		RegFuncsGw:         make([]GwRegFunc, 0),
		GrpcDialOptions:    make([]grpc.DialOption, 0),
		ServerMuxOptions:   make([]runtime.ServeMuxOption, 0),
		GwMappingFilePaths: make([]string, 0),
		GwMapping:          make(map[string]*gwRule),
		Mux:                http.NewServeMux(),
	}

	for i := range opts {
		opts[i](entry)
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = "grpcGatewayServer-" + strconv.FormatUint(entry.HttpPort, 10)
	}

	if entry.CommonServiceEntry != nil {
		entry.RegFuncsGw = append(entry.RegFuncsGw, entry.CommonServiceEntry.RegFuncGw)
	}

	return entry
}

// Add grpc dial options.
func (entry *GwEntry) addDialOptions(opts ...grpc.DialOption) {
	entry.GrpcDialOptions = append(entry.GrpcDialOptions, opts...)
}

// Add registration function for gateway.
func (entry *GwEntry) addRegFuncsGw(funcs ...GwRegFunc) {
	entry.RegFuncsGw = append(entry.RegFuncsGw, funcs...)
}

func (entry *GwEntry) parseGwMapping() {
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

func (entry *GwEntry) parseGwMappingHelper(bytes []byte) {
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
		entry.GwMapping[element.GetSelector()] = rule
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

// Bootstrap GwEntry.
func (entry *GwEntry) Bootstrap(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"bootstrap",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	logger := entry.ZapLoggerEntry.GetLogger()

	if raw := ctx.Value(bootstrapEventIdKey); raw != nil {
		event.SetEventId(raw.(string))
		logger = logger.With(zap.String("eventId", event.GetEventId()))
	}

	entry.logBasicInfo(event)

	// Parse gateway mapping file paths.
	entry.parseGwMapping()

	if entry.IsGrpcTlsEnabled() {
		if cert, err := tls.X509KeyPair(entry.GrpcCertEntry.Store.ServerCert, entry.GrpcCertEntry.Store.ServerKey); err != nil {
			rkcommon.ShutdownWithError(err)
		} else {
			tls := credentials.NewTLS(&tls.Config{
				// This is not a good idea, however, grpc-gateway and grpc is running on the same process
				// So it is safe to do this.
				InsecureSkipVerify: true,
				Certificates: []tls.Certificate{cert},
			})
			entry.addDialOptions(grpc.WithTransportCredentials(tls))
		}
	} else {
		entry.addDialOptions(grpc.WithInsecure())
	}

	grpcEndpoint := "0.0.0.0:" + strconv.FormatUint(entry.GrpcPort, 10)

	entry.GwMux = runtime.NewServeMux(entry.ServerMuxOptions...)

	for i := range entry.RegFuncsGw {
		err := entry.RegFuncsGw[i](context.Background(), entry.GwMux, grpcEndpoint, entry.GrpcDialOptions)
		if err != nil {
			entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
			rkcommon.ShutdownWithError(err)
		}
	}

	entry.Mux.Handle("/", entry.GwMux)

	// Is tv enabled?
	if entry.IsTvEnabled() {
		entry.TvEntry.Bootstrap(ctx)
		entry.Mux.HandleFunc("/rk/v1/tv/", entry.TvEntry.TV)
		entry.Mux.HandleFunc("/rk/v1/assets/tv/", entry.TvEntry.AssetsFileHandler)
	}

	// Is swagger enabled?
	if entry.IsSwEnabled() {
		entry.SwEntry.Bootstrap(ctx)
		entry.Mux.HandleFunc(entry.SwEntry.Path, entry.SwEntry.ConfigFileHandler)
		entry.Mux.HandleFunc("/rk/v1/assets/sw/", entry.SwEntry.AssetsFileHandler)
	}

	// Is prom enabled?
	if entry.IsPromEnabled() {
		entry.PromEntry.Bootstrap(ctx)
		// Register prom path into Router.
		entry.Mux.Handle(entry.PromEntry.Path, promhttp.HandlerFor(entry.PromEntry.Gatherer, promhttp.HandlerOpts{}))
	}

	entry.Server = &http.Server{
		Addr:    "0.0.0.0:" + strconv.FormatUint(entry.HttpPort, 10),
		Handler: headMethodHandler(entry.Mux),
	}

	logger.Info("Bootstrapping GwEntry.", event.ListPayloads()...)
	entry.EventLoggerEntry.GetEventHelper().Finish(event)

	go func(*GwEntry) {
		if entry.IsServerTlsEnabled() {
			if cert, err := tls.X509KeyPair(entry.CertEntry.Store.ServerCert, entry.CertEntry.Store.ServerKey); err != nil {
				event.AddErr(err)
				entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing TLS.", event.ListPayloads()...)
				rkcommon.ShutdownWithError(err)
			} else {
				entry.Server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
			}

			if err := entry.Server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				event.AddErr(err)
				entry.ZapLoggerEntry.GetLogger().Error("Error occurs while serving grpc-listener-tls.", event.ListPayloads()...)
				rkcommon.ShutdownWithError(err)
			}
		} else {
			if err := entry.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				event.AddErr(err)
				entry.ZapLoggerEntry.GetLogger().Error("Error occurs while serving grpc-listener.", event.ListPayloads()...)
				rkcommon.ShutdownWithError(err)
			}
		}
	}(entry)
}

// Interrupt GwEntry.
func (entry *GwEntry) Interrupt(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"interrupt",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	logger := entry.ZapLoggerEntry.GetLogger()

	if raw := ctx.Value(bootstrapEventIdKey); raw != nil {
		event.SetEventId(raw.(string))
		logger = logger.With(zap.String("eventId", event.GetEventId()))
	}

	entry.logBasicInfo(event)

	if entry.IsPromEnabled() {
		entry.PromEntry.Interrupt(ctx)
	}

	if entry.IsTvEnabled() {
		entry.TvEntry.Interrupt(ctx)
	}

	if entry.IsSwEnabled() {
		entry.SwEntry.Interrupt(ctx)
	}

	logger.Info("Interrupting gwEntry.", event.ListPayloads()...)

	if entry.Server != nil {
		if err := entry.Server.Shutdown(context.Background()); err != nil {
			event.AddErr(err)
			logger.Warn("Error occurs while stopping gwEntry")
		}
	}

	entry.EventLoggerEntry.GetEventHelper().Finish(event)
}

// Get name of entry.
func (entry *GwEntry) GetName() string {
	return entry.EntryName
}

// Get type of entry.
func (entry *GwEntry) GetType() string {
	return entry.EntryType
}

// Stringfy entry.
func (entry *GwEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// Marshal entry.
func (entry *GwEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":            entry.EntryName,
		"entryType":            entry.EntryType,
		"grpcPort":             entry.GrpcPort,
		"httpPort":             entry.HttpPort,
		"zapLoggerEntry":       entry.ZapLoggerEntry.GetName(),
		"eventLoggerEntry":     entry.EventLoggerEntry.GetName(),
		"swEnabled":            entry.IsSwEnabled(),
		"tvEnabled":            entry.IsTvEnabled(),
		"promEnabled":          entry.IsPromEnabled(),
		"commonServiceEnabled": entry.IsCommonServiceEnabled(),
		"grpcTlsEnabled":       entry.IsGrpcTlsEnabled(),
		"serverTlsEnabled":     entry.IsServerTlsEnabled(),
	}

	return json.Marshal(&m)
}

// Not supported.
func (entry *GwEntry) UnmarshalJSON([]byte) error {
	return nil
}

// Get description of entry.
func (entry *GwEntry) GetDescription() string {
	return entry.EntryDescription
}

// Is swagger enabled?
func (entry *GwEntry) IsSwEnabled() bool {
	return entry.SwEntry != nil
}

// Is tv enabled?
func (entry *GwEntry) IsTvEnabled() bool {
	return entry.TvEntry != nil
}

// Is prometheus client enabled?
func (entry *GwEntry) IsPromEnabled() bool {
	return entry.PromEntry != nil
}

// Is common service enabled?
func (entry *GwEntry) IsCommonServiceEnabled() bool {
	return entry.CommonServiceEntry != nil
}

// Is client TLS enabled?
func (entry *GwEntry) IsGrpcTlsEnabled() bool {
	return entry.GrpcCertEntry != nil && entry.GrpcCertEntry.Store != nil && len(entry.GrpcCertEntry.Store.ServerCert) > 0
}

// Is server TLS enabled?
func (entry *GwEntry) IsServerTlsEnabled() bool {
	return entry.CertEntry != nil && entry.CertEntry.Store != nil && len(entry.CertEntry.Store.ServerCert) > 0
}

// Add basic fields into event.
func (entry *GwEntry) logBasicInfo(event rkquery.Event) {
	event.AddPayloads(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
		zap.Uint64("grpcPort", entry.GrpcPort),
		zap.Uint64("httpPort", entry.HttpPort),
		zap.Bool("swEnabled", entry.IsSwEnabled()),
		zap.Bool("tvEnabled", entry.IsTvEnabled()),
		zap.Bool("promEnabled", entry.IsPromEnabled()),
		zap.Bool("commonServiceEnabled", entry.IsCommonServiceEnabled()),
		zap.Bool("clientTlsEnabled", entry.IsGrpcTlsEnabled()),
		zap.Bool("serverTlsEnabled", entry.IsServerTlsEnabled()))
}

// Support HEAD request.
func headMethodHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			return
		}
		h.ServeHTTP(w, r)
	})
}
