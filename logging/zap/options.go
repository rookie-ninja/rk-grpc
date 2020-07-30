// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_logging_zap

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	DefaultOptions = &Options{
		enableLogging:        EnableLoggingDefault,
		enablePayloadLogging: EnablePayloadLoggingDefault,
		enableMetrics:        EnableMetricsDefault,
		errorToCode:          ErrorToCodesDefault,
	}
)

func MergeOpt(opts []Option) *Options {
	optCopy := &Options{}
	*optCopy = *DefaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

// Default options
func EnableLoggingDefault(string, error) bool {
	return true
}

func EnablePayloadLoggingDefault(string, error) bool {
	return false
}

func EnableMetricsDefault(string, error) bool {
	return true
}

func ErrorToCodesDefault(err error) codes.Code {
	return status.Code(err)
}

type Options struct {
	enableMetrics        Enable
	enableLogging        Enable
	enablePayloadLogging Enable
	errorToCode          Codes
}

type Option func(*Options)

// Implement this if want to enable any functionality among interceptor
type Enable func(method string, err error) bool

type Codes func(err error) codes.Code

func EnableLogging(f Enable) Option {
	return func(o *Options) {
		o.enableLogging = f
	}
}

func EnableMetrics(f Enable) Option {
	return func(o *Options) {
		o.enableMetrics = f
	}
}

func EnablePayloadLogging(f Enable) Option {
	return func(o *Options) {
		o.enablePayloadLogging = f
	}
}

func ErrorToCode(f Codes) Option {
	return func(o *Options) {
		o.errorToCode = f
	}
}
