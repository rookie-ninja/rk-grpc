package rkgrpcextension

import (
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
)

var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:    rkgrpcbasic.RkEntryNameValue,
		EntryType:    rkgrpcbasic.RkEntryTypeValue,
		Prefix:       "RK",
		RequestIdKey: "X-RK-Request-Id",
		TraceIdKey:   "X-RK-Trace-Id",
	}

	for i := range opts {
		opts[i](set)
	}

	if len(set.Prefix) < 1 {
		set.Prefix = "RK"
	}

	set.TraceIdKey = fmt.Sprintf("X-%s-Trace-Id", set.Prefix)
	set.RequestIdKey = fmt.Sprintf("X-%s-Request-Id", set.Prefix)
	rkgrpcctx.RequestIdMetadataKey = set.RequestIdKey

	set.AppNameKey = fmt.Sprintf("X-%s-App-Name", set.Prefix)
	set.AppNameValue = rkentry.GlobalAppCtx.GetAppInfoEntry().AppName
	set.AppVersionKey = fmt.Sprintf("X-%s-App-Version", set.Prefix)
	set.AppVersionValue = rkentry.GlobalAppCtx.GetAppInfoEntry().Version
	set.AppUnixTimeKey = fmt.Sprintf("X-%s-App-Unix-Time", set.Prefix)
	set.ReceivedTimeKey = fmt.Sprintf("X-%s-Received-Time", set.Prefix)

	key := rkgrpcbasic.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing extension interceptor
type optionSet struct {
	EntryName       string
	EntryType       string
	Prefix          string
	RequestIdKey    string
	TraceIdKey      string
	LocationKey     string
	AppNameKey      string
	AppNameValue    string
	AppVersionKey   string
	AppVersionValue string
	AppUnixTimeKey  string
	ReceivedTimeKey string
}

type Option func(*optionSet)

func WithEntryNameAndType(entryName, entryType string) Option {
	return func(opt *optionSet) {
		opt.EntryName = entryName
		opt.EntryType = entryType
	}
}

func WithPrefix(prefix string) Option {
	return func(opt *optionSet) {
		opt.Prefix = prefix
	}
}
