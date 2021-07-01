// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcpanic

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
)

// Interceptor would distinguish entry.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type nad options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcinter.RpcEntryNameValue,
		EntryType: rkgrpcinter.RpcEntryTypeValue,
	}

	for i := range opts {
		opts[i](set)
	}

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// Options which is used while initializing logging interceptor
type optionSet struct {
	EntryName string
	EntryType string
}

type Option func(*optionSet)

// Provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}
