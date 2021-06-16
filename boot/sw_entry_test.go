// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithNameSw_HappyCase(t *testing.T) {
	opt := WithNameSw("ut-name")
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, "ut-name", entry.GetName())
}

func TestWithPortSw_HappyCase(t *testing.T) {
	opt := WithPortSw(1234)
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, uint64(1234), entry.Port)
}

func TestWithPathSw_HappyCase(t *testing.T) {
	opt := WithPathSw("ut-path")
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, "/ut-path/", entry.Path)
}

func TestWithJsonPathSw_HappyCase(t *testing.T) {
	opt := WithJsonPathSw("ut-json-path")
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, "ut-json-path", entry.JsonPath)
}

func TestWithHeadersSw_HappyCase(t *testing.T) {
	m := map[string]string{
		"key": "value",
	}
	opt := WithHeadersSw(m)
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, m, entry.Headers)
}

func TestWithZapLoggerEntrySw_HappyCase(t *testing.T) {
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()
	opt := WithZapLoggerEntrySw(zapLoggerEntry)
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, zapLoggerEntry, entry.ZapLoggerEntry)
}

func TestWithEventLoggerEntrySw_HappyCase(t *testing.T) {
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
	opt := WithEventLoggerEntrySw(eventLoggerEntry)
	entry := NewSwEntry()

	opt(entry)
	assert.Equal(t, eventLoggerEntry, entry.EventLoggerEntry)
}

func TestWithEnableCommonServiceSw_HappyCase(t *testing.T) {
	// Expect true
	opt := WithEnableCommonServiceSw(true)
	entry := NewSwEntry()
	opt(entry)
	assert.True(t, entry.EnableCommonService)

	// Expect false
	opt = WithEnableCommonServiceSw(false)
	opt(entry)
	assert.False(t, entry.EnableCommonService)
}

func TestNewSwEntry_HappyCase(t *testing.T) {
	entry := NewSwEntry(
		WithNameSw("ut-sw"),
		WithPortSw(1234),
		WithEnableCommonServiceSw(true))

	assert.Equal(t, "ut-sw", entry.GetName())
	assert.Equal(t, SwEntryType, entry.GetType())
	assert.Equal(t, SwEntryDescription, entry.GetDescription())
	assert.Equal(t, "/sw/", entry.Path)
	assert.NotNil(t, entry.ZapLoggerEntry)
	assert.NotNil(t, entry.EventLoggerEntry)

	assert.NotEmpty(t, swConfigFileContents)
	jsonFileContent, ok := swaggerJsonFiles["ut-sw"+SwEntryCommonServiceJsonFileSuffix]
	assert.True(t, ok)
	assert.NotEmpty(t, jsonFileContent)

	assert.NotEmpty(t, commonServiceJson)
	assert.NotEmpty(t, swaggerIndexHTML)
}

func TestSwEntry_Bootstrap_HappyCase(t *testing.T) {
	assertNotPanic(t)
	entry := NewSwEntry(
		WithZapLoggerEntrySw(rkentry.NoopZapLoggerEntry()),
		WithEventLoggerEntrySw(rkentry.NoopEventLoggerEntry()))
	entry.Bootstrap(context.TODO())
}

func TestSwEntry_Interrupt_HappyCase(t *testing.T) {
	assertNotPanic(t)
	entry := NewSwEntry(
		WithZapLoggerEntrySw(rkentry.NoopZapLoggerEntry()),
		WithEventLoggerEntrySw(rkentry.NoopEventLoggerEntry()))
	entry.Interrupt(context.TODO())
}

func TestSwEntry_GetName_HappyCase(t *testing.T) {
	entry := NewSwEntry(
		WithNameSw("ut-sw"))
	assert.Equal(t, "ut-sw", entry.GetName())
}

func TestSwEntry_GetType_HappyCase(t *testing.T) {
	entry := NewSwEntry()
	assert.Equal(t, SwEntryType, entry.GetType())
}

func TestSwEntry_String_HappyCase(t *testing.T) {
	entry := NewSwEntry()

	str := entry.String()
	assert.Contains(t, str, "entryName")
	assert.Contains(t, str, "entryType")
	assert.Contains(t, str, "entryDescription")
	assert.Contains(t, str, "eventLoggerEntry")
	assert.Contains(t, str, "zapLoggerEntry")
	assert.Contains(t, str, str, "jsonPath")
	assert.Contains(t, str, "port")
	assert.Contains(t, str, "path")
	assert.Contains(t, str, "headers")
}

func TestSwEntry_GetDescription_HappyCase(t *testing.T) {
	entry := NewSwEntry()

	assert.Equal(t, SwEntryDescription, entry.GetDescription())
}

func TestSwEntry_MarshalJSON_HappyCase(t *testing.T) {
	entry := NewSwEntry()

	bytes, err := entry.MarshalJSON()
	assert.Nil(t, err)
	assert.NotEmpty(t, bytes)

	str := string(bytes)
	assert.Contains(t, str, "entryName")
	assert.Contains(t, str, "entryType")
	assert.Contains(t, str, "entryDescription")
	assert.Contains(t, str, "eventLoggerEntry")
	assert.Contains(t, str, "zapLoggerEntry")
	assert.Contains(t, str, str, "jsonPath")
	assert.Contains(t, str, "port")
	assert.Contains(t, str, "path")
	assert.Contains(t, str, "headers")
}

func TestSwEntry_UnmarshalJSON_HappyCase(t *testing.T) {
	entry := NewSwEntry()
	assert.Nil(t, entry.UnmarshalJSON(nil))
}

func TestSwEntry_logBasicInfo_HappyCase(t *testing.T) {
	entry := NewSwEntry()
	event := rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEvent()

	entry.logBasicInfo(event)
	fields := event.ListPayloads()

	assert.Len(t, fields, 4)
}

func TestSwEntry_initSwaggerConfig_HappyCase(t *testing.T) {
	entry := SwEntry{
		EntryName:           SwEntryNameDefault,
		EntryType:           SwEntryType,
		EntryDescription:    SwEntryDescription,
		EnableCommonService: true,
		ZapLoggerEntry:      rkentry.NoopZapLoggerEntry(),
		EventLoggerEntry:    rkentry.NoopEventLoggerEntry(),
		Port:                1234,
		Path:                "/sw/",
	}

	entry.initSwaggerConfig()

	assert.Equal(t, SwEntryNameDefault, entry.GetName())
	assert.Equal(t, SwEntryType, entry.GetType())
	assert.Equal(t, SwEntryDescription, entry.GetDescription())
	assert.Equal(t, "/sw/", entry.Path)
	assert.NotNil(t, entry.ZapLoggerEntry)
	assert.NotNil(t, entry.EventLoggerEntry)

	assert.NotEmpty(t, swConfigFileContents)
	jsonFileContent, ok := swaggerJsonFiles["ut-sw"+SwEntryCommonServiceJsonFileSuffix]
	assert.True(t, ok)
	assert.NotEmpty(t, jsonFileContent)

	assert.NotEmpty(t, commonServiceJson)
	assert.NotEmpty(t, swaggerIndexHTML)
}
