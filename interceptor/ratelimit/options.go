// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpclimit is a middleware of grpc framework for rate limiting.
package rkgrpclimit

import (
	"context"
	juju "github.com/juju/ratelimit"
	"github.com/rookie-ninja/rk-common/error"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	uber "go.uber.org/ratelimit"
	"strings"
	"time"
)

const (
	TokenBucket   = "tokenBucket"
	LeakyBucket   = "leakyBucket"
	DefaultLimit  = 1000000
	GlobalLimiter = "rk-limiter"
)

// User could implement
type Limiter func(ctx context.Context) error

// NoopLimiter will do nothing
type NoopLimiter struct{}

// Limit will do nothing
func (l *NoopLimiter) Limit(context.Context) error {
	return nil
}

// ZeroRateLimiter will block requests.
type ZeroRateLimiter struct{}

// Limit will block request and return error
func (l *ZeroRateLimiter) Limit(context.Context) error {
	return rkerror.ResourceExhausted("Slow down your request.").Err()
}

// tokenBucketLimiter delegates limit logic to juju.Bucket
type tokenBucketLimiter struct {
	delegator *juju.Bucket
}

// Limit delegates limit logic to juju.Bucket
func (l *tokenBucketLimiter) Limit(ctx context.Context) error {
	l.delegator.Wait(1)
	return nil
}

// leakyBucketLimiter delegates limit logic to uber.Limiter
type leakyBucketLimiter struct {
	delegator uber.Limiter
}

// Limit delegates limit logic to uber.Limiter
func (l *leakyBucketLimiter) Limit(ctx context.Context) error {
	l.delegator.Take()
	return nil
}

// Interceptor would distinguish auth set based on.
var optionsMap = make(map[string]*optionSet)

// Create new optionSet with rpc type and options.
func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:       rkgrpcinter.RpcEntryNameValue,
		EntryType:       rkgrpcinter.RpcEntryTypeValue,
		reqPerSec:       DefaultLimit,
		reqPerSecByPath: make(map[string]int, DefaultLimit),
		algorithm:       TokenBucket,
		limiter:         make(map[string]Limiter),
	}

	for i := range opts {
		opts[i](set)
	}

	switch set.algorithm {
	case TokenBucket:
		if set.reqPerSec < 1 {
			l := &ZeroRateLimiter{}
			set.setLimiter(GlobalLimiter, l.Limit)
		} else {
			l := &tokenBucketLimiter{
				delegator: juju.NewBucketWithRate(float64(set.reqPerSec), int64(set.reqPerSec)),
			}
			set.setLimiter(GlobalLimiter, l.Limit)
		}

		for k, v := range set.reqPerSecByPath {
			if v < 1 {
				l := &ZeroRateLimiter{}
				set.setLimiter(k, l.Limit)
			} else {
				l := &tokenBucketLimiter{
					delegator: juju.NewBucketWithRate(float64(v), int64(v)),
				}
				set.setLimiter(k, l.Limit)
			}
		}
	case LeakyBucket:
		if set.reqPerSec < 1 {
			l := &ZeroRateLimiter{}
			set.setLimiter(GlobalLimiter, l.Limit)
		} else {
			l := &leakyBucketLimiter{
				delegator: uber.New(set.reqPerSec),
			}
			set.setLimiter(GlobalLimiter, l.Limit)
		}

		for k, v := range set.reqPerSecByPath {
			if v < 1 {
				l := &ZeroRateLimiter{}
				set.setLimiter(k, l.Limit)
			} else {
				l := &leakyBucketLimiter{
					delegator: uber.New(v),
				}
				set.setLimiter(k, l.Limit)
			}
		}
	default:
		l := &NoopLimiter{}
		set.setLimiter(GlobalLimiter, l.Limit)
	}

	key := rkgrpcinter.ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing extension interceptor
type optionSet struct {
	EntryName       string
	EntryType       string
	reqPerSec       int
	reqPerSecByPath map[string]int
	algorithm       string
	limiter         map[string]Limiter
}

// Wait until rate limit pass through
func (set *optionSet) Wait(ctx context.Context, path string) (time.Duration, error) {
	now := time.Now()

	limiter := set.getLimiter(path)
	if err := limiter(ctx); err != nil {
		return now.Sub(now), err
	}

	return now.Sub(time.Now()), nil
}

func (set *optionSet) getLimiter(path string) Limiter {
	if v, ok := set.limiter[path]; ok {
		return v
	}

	return set.limiter[GlobalLimiter]
}

// Set limiter if not exists
func (set *optionSet) setLimiter(path string, l Limiter) {
	if _, ok := set.limiter[path]; ok {
		return
	}

	set.limiter[path] = l
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

// WithReqPerSec Provide request per second.
func WithReqPerSec(reqPerSec int) Option {
	return func(opt *optionSet) {
		if reqPerSec >= 0 {
			opt.reqPerSec = reqPerSec
		}
	}
}

// WithReqPerSecByPath Provide request per second by path.
func WithReqPerSecByPath(path string, reqPerSec int) Option {
	return func(opt *optionSet) {
		if reqPerSec >= 0 {
			opt.reqPerSecByPath[path] = reqPerSec
		}
	}
}

// WithAlgorithm provide algorithm of rate limit.
// - tokenBucket
// - leakyBucket
func WithAlgorithm(algo string) Option {
	return func(opt *optionSet) {
		opt.algorithm = algo
	}
}

// WithGlobalLimiter provide user defined Limiter.
func WithGlobalLimiter(l Limiter) Option {
	return func(opt *optionSet) {
		opt.limiter[GlobalLimiter] = l
	}
}

// WithLimiterByPath provide user defined Limiter by path.
func WithLimiterByPath(path string, l Limiter) Option {
	return func(opt *optionSet) {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		opt.limiter[path] = l
	}
}
