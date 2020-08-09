// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_inter_logging

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	DefaultOptions = &Options{
		enableLogging:        EnableLogging,
		enablePayloadLogging: EnablePayloadLogging,
		enableMetrics:        EnableMetrics,
		errorToCode:          ErrorToCodes,
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
func DisableLogging() bool {
	return false
}

func EnableLogging() bool {
	return true
}

func EnablePayloadLogging() bool {
	return true
}

func DisablePayloadLogging() bool {
	return false
}

func EnableMetrics() bool {
	return true
}

func DisableMetrics() bool {
	return false
}

func ErrorToCodes(err error) codes.Code {
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
type Enable func() bool

type Codes func(err error) codes.Code

func EnableLoggingOption(f Enable) Option {
	return func(o *Options) {
		o.enableLogging = f
	}
}

func EnableMetricsOption(f Enable) Option {
	return func(o *Options) {
		o.enableMetrics = f
	}
}

func EnablePayloadLoggingOption(f Enable) Option {
	return func(o *Options) {
		o.enablePayloadLogging = f
	}
}

func ErrorToCodeOption(f Codes) Option {
	return func(o *Options) {
		o.errorToCode = f
	}
}
