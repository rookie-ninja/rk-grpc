// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import "github.com/rookie-ninja/rk-grpc/interceptor/context"

// Interceptor would distinguish metrics set based on.
var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcctx.RkEntryNameValue,
		EntryType: rkgrpcctx.RkEntryTypeValue,
	}

	for i := range opts {
		opts[i](set)
	}

	key := rkgrpcctx.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing basic interceptor
type optionSet struct {
	EntryName string
	EntryType string
}

type Option func(*optionSet)

func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		if len(entryName) > 0 {
			set.EntryName = entryName
		}
		if len(entryType) > 0 {
			set.EntryType = entryType
		}
	}
}
