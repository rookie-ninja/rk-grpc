// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc_retry

import (
	"math/rand"
	"time"
)

// JitterUp adds random jitter to the duration.
//
// This adds or substracts time from the duration within a given jitter fraction.
// For example for 10s and jitter 0.1, it will returna  time within [9s, 11s])
func JitterUp(duration time.Duration, jitter float64) time.Duration {
	multiplier := jitter * (rand.Float64()*2 - 1)
	return time.Duration(float64(duration) * (1 + multiplier))
}

// ExponentBase2 computes 2^(a-1) where a >= 1. If a is 0, the result is 0.
func ExponentBase2(a uint) uint {
	return (1 << a) >> 1
}

// BackoffLinear is very simple: it waits for a fixed period of time between calls.
func BackoffLinear(waitBetween time.Duration) BackoffFunc {
	return func(attempt uint) time.Duration {
		return waitBetween
	}
}

// BackoffLinearWithJitter waits a set period of time, allowing for jitter (fractional adjustment).
//
// For example waitBetween=1s and jitter=0.10 can generate waits between 900ms and 1100ms.
func BackoffLinearWithJitter(waitBetween time.Duration, jitterFraction float64) BackoffFunc {
	return func(attempt uint) time.Duration {
		return JitterUp(waitBetween, jitterFraction)
	}
}

// BackoffExponential produces increasing intervals for each attempt.
//
// The scalar is multiplied times 2 raised to the current attempt. So the first
// retry with a scalar of 100ms is 100ms, while the 5th attempt would be 3.2s.
func BackoffExponential(scalar time.Duration) BackoffFunc {
	return func(attempt uint) time.Duration {
		return scalar * time.Duration(ExponentBase2(attempt))
	}
}

// BackoffExponentialWithJitter creates an exponential backoff like
// BackoffExponential does, but adds jitter.
func BackoffExponentialWithJitter(scalar time.Duration, jitterFraction float64) BackoffFunc {
	return func(attempt uint) time.Duration {
		return JitterUp(scalar*time.Duration(ExponentBase2(attempt)), jitterFraction)
	}
}
