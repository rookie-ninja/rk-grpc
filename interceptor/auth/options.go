// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcauth

import (
	"encoding/base64"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"path"
	"strings"
)

const (
	typeBasic  = "Basic"
	typeApiKey = "X-API-Key"
)

// Interceptor would distinguish auth set based on.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type nad options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:     rkgrpcinter.RpcEntryNameValue,
		EntryType:     rkgrpcinter.RpcEntryTypeValue,
		BasicRealm:    "",
		BasicAccounts: make(map[string]bool),
		ApiKey:        make(map[string]bool),
		IgnorePrefix:  make([]string, 0),
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
	EntryName     string
	EntryType     string
	BasicRealm    string
	BasicAccounts map[string]bool
	ApiKey        map[string]bool
	IgnorePrefix  []string
}

func (set *optionSet) ShouldAuth(method string) bool {
	if len(set.BasicAccounts) < 1 && len(set.ApiKey) < 1 {
		return false
	}

	for i := range set.IgnorePrefix {
		if strings.HasPrefix(method, set.IgnorePrefix[i]) {
			return false
		}
	}

	return true
}

// Check permission with username and password.
func (set *optionSet) Authorized(authType, cred string) bool {
	switch authType {
	case typeBasic:
		_, ok := set.BasicAccounts[cred]
		return ok
	case typeApiKey:
		_, ok := set.ApiKey[cred]
		return ok
	}

	return false
}

type Option func(*optionSet)

// Provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		set.EntryName = entryName
		set.EntryType = entryType
	}
}

// Provide basic auth credentials formed as user:pass.
// We will encode credential with base64 since incoming credential from client would be encoded.
func WithBasicAuth(cred ...string) Option {
	return func(set *optionSet) {
		for i := range cred {
			set.BasicAccounts[base64.StdEncoding.EncodeToString([]byte(cred[i]))] = true
		}
	}
}

// Provide API Key auth credentials.
// An API key is a token that a client provides when making API calls.
// With API key auth, you send a key-value pair to the API either in the request headers or query parameters.
// Some APIs use API keys for authorization.
//
// The API key was injected into incoming header with key of X-API-Key
func WithApiKeyAuth(key ...string) Option {
	return func(set *optionSet) {
		for i := range key {
			set.ApiKey[key[i]] = true
		}
	}
}

// Provide methods that will ignore.
// Mainly used for swagger main page and RK TV entry.
func WithIgnorePrefix(paths ...string) Option {
	return func(set *optionSet) {
		for i := range paths {
			set.IgnorePrefix = append(set.IgnorePrefix, path.Join("/", paths[i]))
		}
	}
}
