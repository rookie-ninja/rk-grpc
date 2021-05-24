// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpctokenauth

import "github.com/rookie-ninja/rk-grpc/interceptor/context"

// Interceptor would distinguish token set based on.
var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcctx.RkEntryNameValue,
		EntryType: rkgrpcctx.RkEntryTypeValue,
		tokens:    make(map[string]bool),
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

// options which is used while initializing logging interceptor
type optionSet struct {
	EntryName string
	EntryType string
	tokens    map[string]bool
}

func (set *optionSet) Authorized(token string) bool {
	if val, ok := set.tokens[token]; !ok {
		return false
	} else {
		return !val
	}
}

type Option func(*optionSet)

func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}

func WithToken(token string, expired bool) Option {
	return func(set *optionSet) {
		set.tokens[token] = expired
	}
}
