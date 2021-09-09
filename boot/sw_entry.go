// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	"encoding/json"
	"github.com/markbates/pkger"
	"github.com/markbates/pkger/pkging"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	swaggerIndexHTML     = readFileFromPkger("/assets/sw/index.html")
	commonServiceJson    = readFileFromPkger("/api/third_party/gen/v1/rk_common_service.swagger.json")
	swConfigFileContents = ``
	swaggerJsonFiles     = make(map[string]string, 0)
)

const (
	// SwEntryType default entry type
	SwEntryType = "GrpcSwEntry"
	// SwEntryNameDefault default entry name
	SwEntryNameDefault = "GrpcSwDefault"
	// SwEntryDescription default entry description
	SwEntryDescription = "Internal RK entry which implements swagger with Grpc framework."
	// SwEntryCommonServiceJsonFileSuffix default swagger json file suffix
	SwEntryCommonServiceJsonFileSuffix = "-rk-common.swagger.json"
)

// Inner struct used while initializing swagger entry.
type swUrlConfig struct {
	Urls []*swUrl `json:"urls" yaml:"urls"`
}

// Inner struct used while initializing swagger entry.
type swUrl struct {
	Name string `json:"name" yaml:"name"`
	Url  string `json:"url" yaml:"url"`
}

// BootConfigSw Bootstrap config of swagger.
// 1: Enabled: Enable swagger.
// 2: Path: Swagger path accessible from restful API.
// 3: JsonPath: The path of where swagger JSON file was located.
// 4: Headers: The headers that would added into each API response.
type BootConfigSw struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Path     string   `yaml:"path" json:"path"`
	JsonPath string   `yaml:"jsonPath" json:"jsonPath"`
	Headers  []string `yaml:"headers" json:"headers"`
}

// SwEntry implements rkentry.Entry interface.
// 1: Path: Swagger path accessible from restful API.
// 2: JsonPath: The path of where swagger JSON file was located.
// 3: Headers: The headers that would added into each API response.
// 4: Port: The port where swagger would listen to.
type SwEntry struct {
	EntryName           string                    `json:"entryName" yaml:"entryName"`
	EntryType           string                    `json:"entryType" yaml:"entryType"`
	EntryDescription    string                    `json:"entryDescription" yaml:"entryDescription"`
	EventLoggerEntry    *rkentry.EventLoggerEntry `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	ZapLoggerEntry      *rkentry.ZapLoggerEntry   `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	JsonPath            string                    `json:"jsonPath" yaml:"jsonPath"`
	Path                string                    `json:"path" yaml:"path"`
	Headers             map[string]string         `json:"headers" yaml:"headers"`
	Port                uint64                    `json:"port" yaml:"port"`
	EnableCommonService bool                      `json:"enableCommonService" yaml:"enableCommonService"`
}

// SwOption Swagger entry option.
type SwOption func(*SwEntry)

// WithNameSw Provide name.
func WithNameSw(name string) SwOption {
	return func(entry *SwEntry) {
		entry.EntryName = name
	}
}

// WithPortSw Provide port.
func WithPortSw(port uint64) SwOption {
	return func(entry *SwEntry) {
		entry.Port = port
	}
}

// WithPathSw Provide path.
func WithPathSw(path string) SwOption {
	return func(entry *SwEntry) {
		if len(path) < 1 {
			path = "/sw/"
		}

		entry.Path = path

		if !strings.HasPrefix(entry.Path, "/") {
			entry.Path = "/" + entry.Path
		}

		if !strings.HasSuffix(entry.Path, "/") {
			entry.Path = entry.Path + "/"
		}
	}
}

// WithJsonPathSw Provide JsonPath.
func WithJsonPathSw(path string) SwOption {
	return func(entry *SwEntry) {
		entry.JsonPath = path
	}
}

// WithHeadersSw Provide headers.
func WithHeadersSw(headers map[string]string) SwOption {
	return func(entry *SwEntry) {
		entry.Headers = headers
	}
}

// WithZapLoggerEntrySw Provide rkentry.ZapLoggerEntry.
func WithZapLoggerEntrySw(zapLoggerEntry *rkentry.ZapLoggerEntry) SwOption {
	return func(entry *SwEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// WithEventLoggerEntrySw Provide rkentry.EventLoggerEntry.
func WithEventLoggerEntrySw(eventLoggerEntry *rkentry.EventLoggerEntry) SwOption {
	return func(entry *SwEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// WithEnableCommonServiceSw Provide rkentry.EventLoggerEntry.
func WithEnableCommonServiceSw(enabled bool) SwOption {
	return func(entry *SwEntry) {
		entry.EnableCommonService = enabled
	}
}

// NewSwEntry Create new swagger entry with options.
func NewSwEntry(opts ...SwOption) *SwEntry {
	entry := &SwEntry{
		EntryName:        SwEntryNameDefault,
		EntryType:        SwEntryType,
		EntryDescription: SwEntryDescription,
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		Path:             "sw",
	}

	for i := range opts {
		opts[i](entry)
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = "grpcSw-" + strconv.FormatUint(entry.Port, 10)
	}

	if !strings.HasPrefix(entry.Path, "/") {
		entry.Path = "/" + entry.Path
	}

	if !strings.HasSuffix(entry.Path, "/") {
		entry.Path = entry.Path + "/"
	}

	// init swagger configs
	entry.initSwaggerConfig()

	return entry
}

// Bootstrap swagger entry.
func (entry *SwEntry) Bootstrap(ctx context.Context) {
	// No op
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

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	logger.Info("Bootstrapping SwEntry.", event.ListPayloads()...)
}

// Interrupt swagger entry.
func (entry *SwEntry) Interrupt(ctx context.Context) {
	// No op
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

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	logger.Info("Interrupting SwEntry.", event.ListPayloads()...)
}

// GetName Get name of entry.
func (entry *SwEntry) GetName() string {
	return entry.EntryName
}

// GetType Get type of entry.
func (entry *SwEntry) GetType() string {
	return entry.EntryType
}

// String Stringfy swagger entry
func (entry *SwEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// GetDescription Get description of entry.
func (entry *SwEntry) GetDescription() string {
	return entry.EntryDescription
}

// MarshalJSON Marshal entry
func (entry *SwEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":        entry.EntryName,
		"entryType":        entry.EntryType,
		"entryDescription": entry.EntryDescription,
		"eventLoggerEntry": entry.EventLoggerEntry.GetName(),
		"zapLoggerEntry":   entry.ZapLoggerEntry.GetName(),
		"jsonPath":         entry.JsonPath,
		"port":             entry.Port,
		"path":             entry.Path,
		"headers":          entry.Headers,
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Unmarshal entry
func (entry *SwEntry) UnmarshalJSON([]byte) error {
	return nil
}

// Add basic fields into event
func (entry *SwEntry) logBasicInfo(event rkquery.Event) {
	event.AddPayloads(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
		zap.String("swPath", entry.Path),
		zap.Any("headers", entry.Headers))
}

// Init swagger config.
// This function do the things bellow:
// 1: List swagger files from entry.JSONPath.
// 2: Read user swagger json files and deduplicate.
// 3: Assign swagger contents into swaggerConfigJson variable
func (entry *SwEntry) initSwaggerConfig() {
	swaggerUrlConfig := &swUrlConfig{
		Urls: make([]*swUrl, 0),
	}

	// 1: Add user API swagger JSON
	entry.listFilesWithSuffix(swaggerUrlConfig)

	// 2: Add rk common APIs
	if entry.EnableCommonService {
		key := entry.EntryName + SwEntryCommonServiceJsonFileSuffix
		// add common service json file
		swaggerJsonFiles[key] = string(commonServiceJson)
		swaggerUrlConfig.Urls = append(swaggerUrlConfig.Urls, &swUrl{
			Name: key,
			Url:  path.Join(entry.Path, key),
		})
	}

	// 3: Marshal to swagger-config.json and write to pkger
	bytes, err := json.Marshal(swaggerUrlConfig)
	if err != nil {
		entry.ZapLoggerEntry.GetLogger().Error("Failed to unmarshal swagger-config.json",
			zap.Error(err))
		rkcommon.ShutdownWithError(err)
	}

	swConfigFileContents = string(bytes)
}

// List files with .json suffix and store them into swaggerJsonFiles variable.
func (entry *SwEntry) listFilesWithSuffix(urlConfig *swUrlConfig) {
	jsonPath := entry.JsonPath
	suffix := ".json"
	// re-path it with working directory if not absolute path
	if !path.IsAbs(entry.JsonPath) {
		wd, err := os.Getwd()
		if err != nil {
			entry.ZapLoggerEntry.GetLogger().Info("Failed to get working directory",
				zap.String("error", err.Error()))
			rkcommon.ShutdownWithError(err)
		}
		jsonPath = path.Join(wd, jsonPath)
	}

	files, err := ioutil.ReadDir(jsonPath)
	if err != nil {
		entry.ZapLoggerEntry.GetLogger().Error("Failed to list files with suffix",
			zap.String("path", jsonPath),
			zap.String("suffix", suffix),
			zap.String("error", err.Error()))
		rkcommon.ShutdownWithError(err)
	}

	for i := range files {
		file := files[i]
		if !file.IsDir() && strings.HasSuffix(file.Name(), suffix) {
			bytes, err := ioutil.ReadFile(path.Join(jsonPath, file.Name()))
			key := entry.EntryName + "-" + file.Name()

			if err != nil {
				entry.ZapLoggerEntry.GetLogger().Info("Failed to read file with suffix",
					zap.String("path", path.Join(jsonPath, key)),
					zap.String("suffix", suffix),
					zap.String("error", err.Error()))
				rkcommon.ShutdownWithError(err)
			}

			swaggerJsonFiles[key] = string(bytes)

			urlConfig.Urls = append(urlConfig.Urls, &swUrl{
				Name: key,
				Url:  path.Join(entry.Path, key),
			})
		}
	}
}

// AssetsFileHandler Http handler which handles assets files of swagger web UI with path prefix of /rk/v1/*
func (entry *SwEntry) AssetsFileHandler(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/rk/v1"), "/")

	if file, err := pkger.Open(path.Join("/boot", p)); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	} else {
		http.ServeContent(w, r, path.Base(p), time.Now(), file)
	}
}

// ConfigFileHandler Http handler which handles swagger.json files with prefix of SwEntry.Path/*
func (entry *SwEntry) ConfigFileHandler(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSuffix(r.URL.Path, "/")

	w.Header().Set("cache-control", "no-cache")

	for k, v := range entry.Headers {
		w.Header().Set(k, v)
	}

	switch p {
	case "/sw":
		if file, err := pkger.Open("/boot/assets/sw/index.html"); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		} else {
			http.ServeContent(w, r, "index.html", time.Now(), file)
		}
	case "/sw/swagger-config.json":
		http.ServeContent(w, r, "swagger-config.json", time.Now(), strings.NewReader(swConfigFileContents))
	default:
		p = strings.TrimPrefix(p, "/sw/")
		value, ok := swaggerJsonFiles[p]

		if ok {
			http.ServeContent(w, r, p, time.Now(), strings.NewReader(value))
		} else {
			http.NotFound(w, r)
		}
	}
}

// Read go template files with Pkger.
func readFileFromPkger(filePath string) []byte {
	var file pkging.File
	var err error

	if file, err = pkger.Open(path.Join("/boot", filePath)); err != nil {
		return []byte{}
	}

	var bytes []byte
	if bytes, err = ioutil.ReadAll(file); err != nil {
		return []byte{}
	}

	return bytes
}
