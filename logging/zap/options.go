package rk_logging_zap

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ServerOpts = []Option{
		WithLog(EnableLog),
		WithPayload(EnablePayload),
		WithProm(EnableProm),
		WithCodes(Codes),
	}

	ClientOpts = []Option{
		WithLog(EnableLog),
		WithPayload(EnablePayload),
		WithProm(EnableProm),
		WithCodes(Codes),
	}

	DefaultOptions = &Options{
		enableLog:     EnableLog,
		enablePayload: EnablePayload,
		enableProm:    EnableProm,
		codeFunc:      Codes,
	}
)

func EvaluateServerOpt(opts []Option) *Options {
	optCopy := &Options{}
	*optCopy = *DefaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

func EvaluateClientOpt(opts []Option) *Options {
	optCopy := &Options{}
	*optCopy = *DefaultOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

func Codes(err error) codes.Code {
	return status.Code(err)
}

func EnableProm(name string, err error) bool {
	return true
}

func EnableLog(name string, err error) bool {
	return true
}

func EnablePayload(name string, err error) bool {
	return false
}

type Options struct {
	enableProm    Enable
	enableLog     Enable
	enablePayload Enable
	codeFunc      ErrorToCode
}

type Option func(*Options)

// Implement this if want to enable any functionality among interceptor
type Enable func(method string, err error) bool

func WithLog(f Enable) Option {
	return func(o *Options) {
		o.enableLog = f
	}
}

func WithProm(f Enable) Option {
	return func(o *Options) {
		o.enableProm = f
	}
}

func WithPayload(f Enable) Option {
	return func(o *Options) {
		o.enableProm = f
	}
}

type ErrorToCode func(err error) codes.Code

func WithCodes(f ErrorToCode) Option {
	return func(o *Options) {
		o.codeFunc = f
	}
}
