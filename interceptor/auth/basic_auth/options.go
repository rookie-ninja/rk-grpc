// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasicauth

import (
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"strings"
)

// Interceptor would distinguish auth set based on.
var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:   rkgrpcctx.RkEntryNameValue,
		EntryType:   rkgrpcctx.RkEntryTypeValue,
		credentials: make(map[string]string),
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
	EntryName   string
	EntryType   string
	credentials map[string]string
}

func (set *optionSet) Authorized(username, password string) bool {
	if val, ok := set.credentials[username]; !ok {
		return false
	} else {
		return val == password
	}
}

type Option func(*optionSet)

func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}

func WithCredential(cred ...string) Option {
	return func(set *optionSet) {
		for i := range cred {
			tokens := strings.Split(cred[i], ":")
			if len(tokens) < 2 {
				return
			}

			set.credentials[tokens[0]] = tokens[1]
		}
	}
}
