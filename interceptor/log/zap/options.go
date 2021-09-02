// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	rkcommon "github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	rklogger "github.com/rookie-ninja/rk-logger"
	rkquery "github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"os"
	"path"
)

const (
	ENCODING_CONSOLE int = 0
	ENCODING_JSON    int = 1
)

// Interceptor would distinguish logs set based on.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type nad options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:             rkgrpcinter.RpcEntryNameValue,
		EntryType:             rkgrpcinter.RpcEntryTypeValue,
		zapLoggerEntry:        rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		eventLoggerEntry:      rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
		ZapLogger:             rkentry.GlobalAppCtx.GetZapLoggerEntryDefault().GetLogger(),
		zapLoggerOutputPath:   make([]string, 0),
		eventLoggerOutputPath: make([]string, 0),
	}

	for i := range opts {
		opts[i](set)
	}

	set.ZapLogger = set.zapLoggerEntry.GetLogger()

	// Override zap logger encoding and output path if provided by user
	// Override encoding type
	if set.zapLoggerEncoding == ENCODING_JSON || len(set.zapLoggerOutputPath) > 0 {
		if set.zapLoggerEncoding == ENCODING_JSON {
			set.zapLoggerEntry.LoggerConfig.Encoding = "json"
		}

		if len(set.zapLoggerOutputPath) > 0 {
			set.zapLoggerEntry.LoggerConfig.OutputPaths = toAbsPath(set.zapLoggerOutputPath...)
		}

		if set.zapLoggerEntry.LumberjackConfig == nil {
			set.zapLoggerEntry.LumberjackConfig = rklogger.NewLumberjackConfigDefault()
		}

		if logger, err := rklogger.NewZapLoggerWithConf(set.zapLoggerEntry.LoggerConfig, set.zapLoggerEntry.LumberjackConfig); err != nil {
			rkcommon.ShutdownWithError(err)
		} else {
			set.ZapLogger = logger
		}
	}

	// Override event logger output path if provided by user
	if len(set.eventLoggerOutputPath) > 0 {
		set.eventLoggerEntry.LoggerConfig.OutputPaths = toAbsPath(set.eventLoggerOutputPath...)
		if set.eventLoggerEntry.LumberjackConfig == nil {
			set.eventLoggerEntry.LumberjackConfig = rklogger.NewLumberjackConfigDefault()
		}
		if logger, err := rklogger.NewZapLoggerWithConf(set.eventLoggerEntry.LoggerConfig, set.eventLoggerEntry.LumberjackConfig); err != nil {
			rkcommon.ShutdownWithError(err)
		} else {
			set.eventLoggerOverride = logger
		}
	}

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

func toAbsPath(p ...string) []string {
	res := make([]string, 0)

	for i := range p {
		if path.IsAbs(p[i]) {
			res = append(res, p[i])
		}

		wd, _ := os.Getwd()
		res = append(res, path.Join(wd, p[i]))
	}

	return res
}

// Options which is used while initializing logging interceptor
type optionSet struct {
	EntryName             string
	EntryType             string
	zapLoggerEntry        *rkentry.ZapLoggerEntry
	eventLoggerEntry      *rkentry.EventLoggerEntry
	ZapLogger             *zap.Logger
	zapLoggerEncoding     int
	eventLoggerEncoding   rkquery.Encoding
	zapLoggerOutputPath   []string
	eventLoggerOutputPath []string
	eventLoggerOverride   *zap.Logger
}

type Option func(*optionSet)

// Provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}

// Provide rkentry.ZapLoggerEntry.
func WithZapLoggerEntry(zapLoggerEntry *rkentry.ZapLoggerEntry) Option {
	return func(set *optionSet) {
		if zapLoggerEntry != nil {
			set.zapLoggerEntry = zapLoggerEntry
		}
	}
}

// Provide rkentry.EventLoggerEntry.
func WithEventLoggerEntry(eventLoggerEntry *rkentry.EventLoggerEntry) Option {
	return func(set *optionSet) {
		if eventLoggerEntry != nil {
			set.eventLoggerEntry = eventLoggerEntry
		}
	}
}

// Provide ZapLoggerEncodingType.
// json or console is supported.
func WithZapLoggerEncoding(ec int) Option {
	return func(set *optionSet) {
		set.zapLoggerEncoding = ec
	}
}

// Provide ZapLogger Output Path.
// Multiple output path could be supported including stdout.
func WithZapLoggerOutputPaths(path ...string) Option {
	return func(set *optionSet) {
		set.zapLoggerOutputPath = append(set.zapLoggerOutputPath, path...)
	}
}

// Provide ZapLoggerEncodingType.
// ENCODING_CONSOLE or ENCODING_JSON is supported.
func WithEventLoggerEncoding(ec int) Option {
	return func(set *optionSet) {
		switch ec {
		case ENCODING_CONSOLE:
			set.eventLoggerEncoding = rkquery.CONSOLE
		case ENCODING_JSON:
			set.eventLoggerEncoding = rkquery.JSON
		default:
			set.eventLoggerEncoding = rkquery.CONSOLE
		}
	}
}

// Provide EventLogger Output Path.
// Multiple output path could be supported including stdout.
func WithEventLoggerOutputPaths(path ...string) Option {
	return func(set *optionSet) {
		set.eventLoggerOutputPath = append(set.eventLoggerOutputPath, path...)
	}
}
