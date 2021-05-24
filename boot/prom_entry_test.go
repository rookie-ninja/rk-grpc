// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-prom"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithPortProm_HappyCase(t *testing.T) {
	entry := NewPromEntry(WithPortProm(1949))

	assert.Equal(t, uint64(1949), entry.Port)
}

func TestWithPathProm_HappyCase(t *testing.T) {
	entry := NewPromEntry(WithPathProm("ut"))

	assert.Equal(t, "/ut", entry.Path)
}

func TestWithZapLoggerEntryProm_HappyCase(t *testing.T) {
	loggerEntry := rkentry.NoopZapLoggerEntry()

	entry := NewPromEntry(WithZapLoggerEntryProm(loggerEntry))

	assert.Equal(t, loggerEntry, entry.ZapLoggerEntry)
}

func TestWithEventLoggerEntryProm_HappyCase(t *testing.T) {
	loggerEntry := rkentry.NoopEventLoggerEntry()

	entry := NewPromEntry(WithEventLoggerEntryProm(loggerEntry))

	assert.Equal(t, loggerEntry, entry.EventLoggerEntry)
}

func TestWithPusherProm_HappyCase(t *testing.T) {
	pusher, _ := rkprom.NewPushGatewayPusher()

	entry := NewPromEntry(WithPusherProm(pusher))

	assert.Equal(t, pusher, entry.Pusher)
}

func TestWithPromRegistryProm_HappyCase(t *testing.T) {
	registry := prometheus.NewRegistry()

	entry := NewPromEntry(WithPromRegistryProm(registry))

	assert.Equal(t, registry, entry.Registry)
}

func TestNewPromEntry_HappyCase(t *testing.T) {
	port := uint64(1949)
	path := "/ut"
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
	pusher, _ := rkprom.NewPushGatewayPusher()
	registry := prometheus.NewRegistry()

	entry := NewPromEntry(
		WithPortProm(port),
		WithPathProm(path),
		WithZapLoggerEntryProm(zapLoggerEntry),
		WithEventLoggerEntryProm(eventLoggerEntry),
		WithPusherProm(pusher),
		WithPromRegistryProm(registry))

	assert.Equal(t, port, entry.Port)
	assert.Equal(t, path, entry.Path)
	assert.Equal(t, zapLoggerEntry, entry.ZapLoggerEntry)
	assert.Equal(t, eventLoggerEntry, entry.EventLoggerEntry)
	assert.Equal(t, pusher, entry.Pusher)
	assert.Equal(t, registry, entry.Registry)
}

func TestPromEntry_GetName_HappyCase(t *testing.T) {
	entry := NewPromEntry()
	assert.Equal(t, PromEntryNameDefault, entry.GetName())
}

func TestPromEntry_GetType_HappyCase(t *testing.T) {
	entry := NewPromEntry()
	assert.Equal(t, PromEntryType, entry.GetType())
}

func TestPromEntry_String_HappyCase(t *testing.T) {
	entry := NewPromEntry()

	str := entry.String()

	assert.Contains(t, str, "entryName")
	assert.Contains(t, str, "entryType")
	assert.Contains(t, str, "entryDescription")
	assert.Contains(t, str, "path")
	assert.Contains(t, str, "port")
}
