// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Interceptor would distinguish logs set based on.
var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcbasic.RkEntryNameValue,
		EntryType: rkgrpcbasic.RkEntryTypeValue,
		ErrorToCodeFunc: func(err error) codes.Code {
			return status.Code(err)
		},
		ZapLoggerEntry:   rkentry.NoopZapLoggerEntry(),
		EventLoggerEntry: rkentry.NoopEventLoggerEntry(),
	}

	for i := range opts {
		opts[i](set)
	}

	key := rkgrpcbasic.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing logging interceptor
type optionSet struct {
	EntryName        string
	EntryType        string
	ErrorToCodeFunc  func(err error) codes.Code
	ZapLoggerEntry   *rkentry.ZapLoggerEntry
	EventLoggerEntry *rkentry.EventLoggerEntry
}

type Option func(*optionSet)

func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}

func WithErrorToCode(errorToCodeFunc func(err error) codes.Code) Option {
	return func(set *optionSet) {
		if errorToCodeFunc != nil {
			set.ErrorToCodeFunc = errorToCodeFunc
		}
	}
}

func WithZapLoggerEntry(zapLoggerEntry *rkentry.ZapLoggerEntry) Option {
	return func(set *optionSet) {
		if zapLoggerEntry != nil {
			set.ZapLoggerEntry = zapLoggerEntry
		}
	}
}

func WithEventLoggerEntry(eventLoggerEntry *rkentry.EventLoggerEntry) Option {
	return func(set *optionSet) {
		if eventLoggerEntry != nil {
			set.EventLoggerEntry = eventLoggerEntry
		}
	}
}
