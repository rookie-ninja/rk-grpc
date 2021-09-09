// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-prom"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"strings"
)

var (
	// Why 1608? It is the year of first telescope was invented
	defaultPort = uint64(1608)
	defaultPath = "/metrics"
)

const (
	// PromEntryType default entry type
	PromEntryType = "GrpcPromEntry"
	// PromEntryNameDefault default entry name
	PromEntryNameDefault = "GrpcPromDefault"
	// PromEntryDescription default entry description
	PromEntryDescription = "Internal RK entry which implements prometheus client with Grpc framework."
)

// BootConfigProm Boot config which is for prom entry.
//
// 1: Path: PromEntry path, /metrics is default value.
// 2: Enabled: Enable prom entry.
// 3: Pusher.Enabled: Enable pushgateway pusher.
// 4: Pusher.IntervalMs: Interval of pushing metrics to remote pushgateway in milliseconds.
// 5: Pusher.JobName: Job name would be attached as label while pushing to remote pushgateway.
// 6: Pusher.RemoteAddress: Pushgateway address, could be form of http://x.x.x.x or x.x.x.x
// 7: Pusher.BasicAuth: Basic auth used to interact with remote pushgateway.
// 8: Pusher.Cert.Ref: Reference of rkentry.CertEntry.
type BootConfigProm struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
	Pusher  struct {
		Enabled       bool   `yaml:"enabled" json:"enabled"`
		JobName       string `yaml:"jobName" json:"jobName"`
		RemoteAddress string `yaml:"remoteAddress" json:"remoteAddress"`
		IntervalMs    int64  `yaml:"IntervalMs" json:"IntervalMs"`
		BasicAuth     string `yaml:"basicAuth" json:"basicAuth"`
		Cert          struct {
			Ref string `yaml:"ref" json:"ref"`
		} `yaml:"cert" json:"cert"`
	} `yaml:"pusher" json:"pusher"`
}

// PromEntry Prometheus entry which implements rkentry.Entry.
//
// 1: Pusher            Periodic pushGateway pusher
// 2: ZapLoggerEntry    rkentry.ZapLoggerEntry
// 3: EventLoggerEntry  rkentry.EventLoggerEntry
// 4: Port              Exposed port by prom entry
// 5: Path              Exposed path by prom entry
// 6: Registry          Prometheus registry
// 7: Registerer        Prometheus registerer
// 8: Gatherer          Prometheus gatherer
type PromEntry struct {
	Pusher           *rkprom.PushGatewayPusher `json:"pushGateWayPusher" yaml:"pushGateWayPusher"`
	EntryName        string                    `json:"entryName" yaml:"entryName"`
	EntryType        string                    `json:"entryType" yaml:"entryType"`
	EntryDescription string                    `json:"entryDescription" yaml:"entryDescription"`
	ZapLoggerEntry   *rkentry.ZapLoggerEntry   `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	EventLoggerEntry *rkentry.EventLoggerEntry `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	Port             uint64                    `json:"port" yaml:"port"`
	Path             string                    `json:"path" yaml:"path"`
	Registry         *prometheus.Registry      `json:"-" yaml:"-"`
	Registerer       prometheus.Registerer     `json:"-" yaml:"-"`
	Gatherer         prometheus.Gatherer       `json:"-" yaml:"-"`
}

// PromEntryOption Prom entry option used while initializing prom entry via code
type PromEntryOption func(*PromEntry)

// WithNameProm Name of prom entry
func WithNameProm(name string) PromEntryOption {
	return func(entry *PromEntry) {
		entry.EntryName = name
	}
}

// WithPortProm Port of prom entry
func WithPortProm(port uint64) PromEntryOption {
	return func(entry *PromEntry) {
		entry.Port = port
	}
}

// WithPathProm Path of prom entry
func WithPathProm(path string) PromEntryOption {
	return func(entry *PromEntry) {
		entry.Path = path
	}
}

// WithZapLoggerEntryProm rkentry.ZapLoggerEntry of prom entry
func WithZapLoggerEntryProm(zapLoggerEntry *rkentry.ZapLoggerEntry) PromEntryOption {
	return func(entry *PromEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// WithEventLoggerEntryProm rkentry.EventLoggerEntry of prom entry
func WithEventLoggerEntryProm(eventLoggerEntry *rkentry.EventLoggerEntry) PromEntryOption {
	return func(entry *PromEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// WithPusherProm PushGateway of prom entry
func WithPusherProm(pusher *rkprom.PushGatewayPusher) PromEntryOption {
	return func(entry *PromEntry) {
		entry.Pusher = pusher
	}
}

// WithPromRegistryProm Provide a new prometheus registry
func WithPromRegistryProm(registry *prometheus.Registry) PromEntryOption {
	return func(entry *PromEntry) {
		if registry != nil {
			entry.Registry = registry
		}
	}
}

// NewPromEntry Create a prom entry with options and add prom entry to rk_ctx.GlobalAppCtx
func NewPromEntry(opts ...PromEntryOption) *PromEntry {
	entry := &PromEntry{
		Port:             defaultPort,
		Path:             defaultPath,
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EntryName:        PromEntryNameDefault,
		EntryType:        PromEntryType,
		EntryDescription: PromEntryDescription,
		Registerer:       prometheus.DefaultRegisterer,
		Gatherer:         prometheus.DefaultGatherer,
	}

	for i := range opts {
		opts[i](entry)
	}

	// Trim space by default
	entry.Path = strings.TrimSpace(entry.Path)

	if len(entry.Path) < 1 {
		// Invalid path, use default one
		entry.Path = defaultPath
	}

	if !strings.HasPrefix(entry.Path, "/") {
		entry.Path = "/" + entry.Path
	}

	if entry.ZapLoggerEntry == nil {
		entry.ZapLoggerEntry = rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()
	}

	if entry.EventLoggerEntry == nil {
		entry.EventLoggerEntry = rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()
	}

	if entry.Registry != nil {
		entry.Registerer = entry.Registry
		entry.Gatherer = entry.Registry
	}

	return entry
}

// Bootstrap Start prometheus client
func (entry *PromEntry) Bootstrap(ctx context.Context) {
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

	// start pusher
	if entry.Pusher != nil {
		entry.Pusher.Start()
	}

	logger.Info("Bootstrapping promEntry.", event.ListPayloads()...)
}

// Interrupt Shutdown prometheus client
func (entry *PromEntry) Interrupt(ctx context.Context) {
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

	if entry.Pusher != nil {
		entry.Pusher.Stop()
	}

	logger.Info("Interrupting promEntry.", event.ListPayloads()...)
}

// GetName Return name of prom entry
func (entry *PromEntry) GetName() string {
	return entry.EntryName
}

// GetType Return type of prom entry
func (entry *PromEntry) GetType() string {
	return entry.EntryType
}

// GetDescription Get description of entry
func (entry *PromEntry) GetDescription() string {
	return entry.EntryDescription
}

// String Stringfy prom entry
func (entry *PromEntry) String() string {
	bytes, _ := json.Marshal(entry)
	return string(bytes)
}

// MarshalJSON Marshal entry
func (entry *PromEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":         entry.EntryName,
		"entryType":         entry.EntryType,
		"entryDescription":  entry.EntryDescription,
		"pushGateWayPusher": entry.Pusher,
		"eventLoggerEntry":  entry.EventLoggerEntry.GetName(),
		"zapLoggerEntry":    entry.ZapLoggerEntry.GetName(),
		"port":              entry.Port,
		"path":              entry.Path,
	}

	return json.Marshal(&m)
}

// UnmarshalJSON Unmarshal entry
func (entry *PromEntry) UnmarshalJSON(b []byte) error {
	return nil
}

// Log basic info into event
func (entry *PromEntry) logBasicInfo(event rkquery.Event) {
	event.AddPayloads(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
		zap.String("path", entry.Path),
		zap.Uint64("port", entry.Port),
	)

	if entry.Pusher != nil {
		event.AddPayloads(
			zap.String("pusherRemoteAddr", entry.Pusher.RemoteAddress),
			zap.Duration("pusherIntervalMs", entry.Pusher.IntervalMs),
			zap.String("pusherJobName", entry.Pusher.JobName),
		)
	}
}

// RegisterCollectors Register collectors in default registry
func (entry *PromEntry) RegisterCollectors(collectors ...prometheus.Collector) error {
	var err error
	for i := range collectors {
		if innerErr := entry.Registerer.Register(collectors[i]); innerErr != nil {
			err = innerErr
		}
	}

	return err
}
