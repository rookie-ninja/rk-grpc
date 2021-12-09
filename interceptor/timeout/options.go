// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpctimeout is a middleware of grpc framework for timing out request.
package rkgrpctimeout

import (
	"github.com/rookie-ninja/rk-grpc/boot/error"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"strings"
	"time"
)

const (
	TokenBucket   = "tokenBucket"
	LeakyBucket   = "leakyBucket"
	DefaultLimit  = 1000000
	GlobalLimiter = "rk-limiter"
)

const global = "rk-global"

var (
	defaultTimeout  = 5 * time.Second
	defaultResponse = rkgrpcerr.Canceled("Request timed out!").Err()
	globalTimeoutRk = &timeoutRk{
		timeout:  defaultTimeout,
		response: defaultResponse,
	}
)

type timeoutRk struct {
	timeout  time.Duration
	response error
}

// Interceptor would distinguish auth set based on.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type and options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: rkgrpcinter.RpcEntryNameValue,
		EntryType: rkgrpcinter.RpcEntryTypeValue,
		timeouts:  make(map[string]*timeoutRk),
	}

	for i := range opts {
		opts[i](set)
	}

	// add global timeout
	set.timeouts[global] = &timeoutRk{
		timeout:  globalTimeoutRk.timeout,
		response: globalTimeoutRk.response,
	}

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing extension interceptor
type optionSet struct {
	EntryName string
	EntryType string
	timeouts  map[string]*timeoutRk
}

// Get timeout instance with path.
// Global one will be returned if no not found.
func (set *optionSet) getTimeoutRk(path string) *timeoutRk {
	if v, ok := set.timeouts[path]; ok {
		return v
	}

	return set.timeouts[global]
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

// WithTimeoutAndResp Provide global timeout and response handler.
// If response is nil, default globalResponse will be assigned
func WithTimeoutAndResp(timeout time.Duration, resp error) Option {
	return func(set *optionSet) {
		if resp == nil {
			resp = defaultResponse
		}

		if timeout == 0 {
			timeout = defaultTimeout
		}

		globalTimeoutRk.timeout = timeout
		globalTimeoutRk.response = resp
	}
}

// WithTimeoutAndRespByPath Provide timeout and response handler by path.
// If response is nil, default globalResponse will be assigned
func WithTimeoutAndRespByPath(path string, timeout time.Duration, resp error) Option {
	return func(set *optionSet) {
		path = normalisePath(path)

		if resp == nil {
			resp = defaultResponse
		}

		if timeout == 0 {
			timeout = defaultTimeout
		}

		set.timeouts[path] = &timeoutRk{
			timeout:  timeout,
			response: resp,
		}
	}
}

func normalisePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}
