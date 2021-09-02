// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, "ut-entry-name", set.EntryName)
	assert.Equal(t, "ut-entry", set.EntryType)
	assert.Equal(t, set,
		optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeUnaryServer)])
}

func TestWithEventLoggerEntry_HappyCase(t *testing.T) {
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithEventLoggerEntry(eventLoggerEntry))

	assert.Equal(t, eventLoggerEntry, set.eventLoggerEntry)
}

func TestWithZapLoggerEntry_HappyCase(t *testing.T) {
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithZapLoggerEntry(zapLoggerEntry))

	assert.Equal(t, zapLoggerEntry, set.zapLoggerEntry)
}
