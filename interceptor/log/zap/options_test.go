// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, "ut-entry-name", set.EntryName)
	assert.Equal(t, "ut-entry", set.EntryType)
	assert.Equal(t, set,
		optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeUnaryServer)])
}

func TestWithEventLoggerEntry_HappyCase(t *testing.T) {
	eventLoggerEntry := rkentry.NoopEventLoggerEntry()
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer,
		WithEventLoggerEntry(eventLoggerEntry))

	assert.Equal(t, eventLoggerEntry, set.EventLoggerEntry)
}

func TestWithZapLoggerEntry_HappyCase(t *testing.T) {
	zapLoggerEntry := rkentry.NoopZapLoggerEntry()
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer,
		WithZapLoggerEntry(zapLoggerEntry))

	assert.Equal(t, zapLoggerEntry, set.ZapLoggerEntry)
}

func TestWithErrorToCode_HappyCase(t *testing.T) {
	errFunc := func(err error) codes.Code {
		return status.Code(err)
	}

	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer,
		WithErrorToCode(errFunc))

	assert.Equal(t,
		reflect.ValueOf(errFunc).Pointer(),
		reflect.ValueOf(set.ErrorToCodeFunc).Pointer())
}
