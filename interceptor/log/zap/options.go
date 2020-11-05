// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc_log

import (
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	defaultOptions = &options{
		enableLogging:        true,
		enablePayloadLogging: true,
		enableMetrics:        true,
		errorToCode:          defaultErrorToCodes,
		eventFactory:         rk_query.NewEventFactory(),
		logger:               rk_logger.NoopLogger,
	}
)

func mergeOpt(opts []Option) {
	for i := range opts {
		opts[i](defaultOptions)
	}
}

func defaultErrorToCodes(err error) codes.Code {
	return status.Code(err)
}

type options struct {
	enableMetrics        bool
	enableLogging        bool
	enablePayloadLogging bool
	errorToCode          func(err error) codes.Code
	eventFactory         *rk_query.EventFactory
	logger               *zap.Logger
}

type Option func(*options)

func WithEventFactory(factory *rk_query.EventFactory) Option {
	return func(opt *options) {
		if factory == nil {
			factory = rk_query.NewEventFactory()
		}
		opt.eventFactory = factory
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(opt *options) {
		if logger == nil {
			logger = rk_logger.NoopLogger
		}
		opt.logger = logger
	}
}

func WithEnableLogging(enable bool) Option {
	return func(opt *options) {
		opt.enableLogging = enable
	}
}

func WithEnableMetrics(enable bool) Option {
	return func(opt *options) {
		opt.enableMetrics = enable
	}
}

func WithEnablePayloadLogging(enable bool) Option {
	return func(opt *options) {
		opt.enablePayloadLogging = enable
	}
}

func WithErrorToCode(funcs func(err error) codes.Code) Option {
	return func(opt *options) {
		opt.errorToCode = funcs
	}
}
