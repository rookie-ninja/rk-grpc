// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc_retry

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	DefaultRetriableCodes = []codes.Code{
		codes.ResourceExhausted,
		codes.Unavailable,
	}

	defaultOption = &retryOption{
		maxRetries:    0, // disabled
		callTimeoutMS: 0, // disabled
		codes:         DefaultRetriableCodes,
		backoffFunc: BackoffFunc(func(attempt uint) time.Duration {
			return BackoffLinearWithJitter(50*time.Millisecond, 0.10)(attempt)
		}),
	}
)

type BackoffFunc func(attempt uint) time.Duration

func WithDisable() RetryCallOption {
	return WithMax(0)
}

func WithMax(maxRetries uint) RetryCallOption {
	return RetryCallOption{applyFunc: func(opt *retryOption) {
		opt.maxRetries = maxRetries
	}}
}

func WithBackoff(bf BackoffFunc) RetryCallOption {
	return RetryCallOption{applyFunc: func(opt *retryOption) {
		opt.backoffFunc = BackoffFunc(func(attempt uint) time.Duration {
			return bf(attempt)
		})
	}}
}

func WithCodes(retryCodes ...codes.Code) RetryCallOption {
	return RetryCallOption{applyFunc: func(opt *retryOption) {
		opt.codes = retryCodes
	}}
}

func WithPerRetryTimeoutMS(timeoutMS time.Duration) RetryCallOption {
	return RetryCallOption{applyFunc: func(opt *retryOption) {
		opt.callTimeoutMS = timeoutMS
	}}
}

type retryOption struct {
	maxRetries    uint
	callTimeoutMS time.Duration
	codes         []codes.Code
	backoffFunc   BackoffFunc
}

type RetryCallOption struct {
	grpc.EmptyCallOption
	applyFunc func(opt *retryOption)
}

func mergeOption(opt *retryOption, retryOptions []RetryCallOption) *retryOption {
	if len(retryOptions) == 0 {
		return opt
	}

	optCopy := &retryOption{}
	*optCopy = *opt
	for _, f := range retryOptions {
		f.applyFunc(optCopy)
	}
	return optCopy
}

func splitCallOptions(callOptions []grpc.CallOption) ([]grpc.CallOption, []RetryCallOption) {
	gRpcCallOptions := make([]grpc.CallOption, 0)
	retryCallOptions := make([]RetryCallOption, 0)
	for _, opt := range callOptions {
		if element, ok := opt.(RetryCallOption); ok {
			retryCallOptions = append(retryCallOptions, element)
		} else {
			gRpcCallOptions = append(gRpcCallOptions, opt)
		}
	}
	return gRpcCallOptions, retryCallOptions
}
