// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpcmeta is a middleware of grpc framework for adding metadata in RPC response
package rkgrpcmeta

import (
	"fmt"
	"github.com/rookie-ninja/rk-grpc/interceptor"
)

// Interceptor would distinguish auth set based on.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type nad options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcinter.RpcEntryNameValue,
		EntryType: rkgrpcinter.RpcEntryTypeValue,
		Prefix:    "RK",
	}

	for i := range opts {
		opts[i](set)
	}

	if len(set.Prefix) < 1 {
		set.Prefix = "RK"
	}

	set.AppNameKey = fmt.Sprintf("X-%s-App-Name", set.Prefix)
	set.AppVersionKey = fmt.Sprintf("X-%s-App-Version", set.Prefix)
	set.AppUnixTimeKey = fmt.Sprintf("X-%s-App-Unix-Time", set.Prefix)
	set.ReceivedTimeKey = fmt.Sprintf("X-%s-Received-Time", set.Prefix)

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
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
	LocationKey     string
	AppNameKey      string
	AppVersionKey   string
	AppUnixTimeKey  string
	ReceivedTimeKey string
}

// Option option for optionSet
type Option func(*optionSet)

// WithEntryNameAndType Provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(opt *optionSet) {
		opt.EntryName = entryName
		opt.EntryType = entryType
	}
}

// WithPrefix Provide prefix.
func WithPrefix(prefix string) Option {
	return func(opt *optionSet) {
		opt.Prefix = prefix
	}
}
