package rkgrpclimit

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType(t *testing.T) {
	defer assertNotPanic(t)

	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
	assert.Len(t, set.limiter, 1)

	// Should be noop limiter
	set.getLimiter("")(context.TODO())
}

func TestWithReqPerSec(t *testing.T) {
	// With non-zero
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithReqPerSec(1))

	assert.Equal(t, 1, set.reqPerSec)
	assert.Len(t, set.limiter, 1)

	// Should be token based limiter
	set.getLimiter("")(context.TODO())

	// With zero
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithReqPerSec(0))

	assert.Equal(t, 0, set.reqPerSec)
	assert.Len(t, set.limiter, 1)

	// should be zero rate limiter which returns error
	assert.NotNil(t, set.getLimiter("")(context.TODO()))
}

func TestWithReqPerSecByPath(t *testing.T) {
	// with non-zero
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithReqPerSecByPath("ut-method", 1))

	assert.Equal(t, 1, set.reqPerSecByPath["ut-method"])
	assert.NotNil(t, set.limiter["ut-method"])

	// Should be token based limiter
	set.getLimiter("ut-method")(context.TODO())

	// With zero
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithReqPerSecByPath("ut-method", 0))

	assert.Equal(t, 0, set.reqPerSecByPath["ut-method"])
	assert.NotNil(t, set.limiter["ut-method"])

	// should be zero rate limiter which returns error
	assert.NotNil(t, set.getLimiter("ut-method")(context.TODO()))
}

func TestWithAlgorithm(t *testing.T) {
	defer assertNotPanic(t)

	// With invalid algorithm
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithAlgorithm("invalid-algo"))

	// should be noop limiter
	assert.Len(t, set.limiter, 1)
	set.getLimiter("")

	// With leaky bucket non zero
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithAlgorithm(LeakyBucket),
		WithReqPerSec(1),
		WithReqPerSecByPath("ut-method", 1))

	// should be leaky bucket
	assert.Len(t, set.limiter, 2)
	set.getLimiter("")(context.TODO())
	set.getLimiter("ut-method")(context.TODO())
}

func TestWithGlobalLimiter(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithGlobalLimiter(func(ctx context.Context) error {
			return fmt.Errorf("ut error")
		}))

	assert.Len(t, set.limiter, 1)
	assert.NotNil(t, set.getLimiter("")(context.TODO()))
}

func TestWithLimiterByMethod(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithLimiterByPath("/ut-method", func(ctx context.Context) error {
			return fmt.Errorf("ut error")
		}))

	assert.Len(t, set.limiter, 2)
	assert.NotNil(t, set.getLimiter("/ut-method")(context.TODO()))
}

func TestOptionSet_Wait(t *testing.T) {
	defer assertNotPanic(t)

	// With user limiter
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithGlobalLimiter(func(context.Context) error {
			return nil
		}))

	set.Wait(context.TODO(), "ut-method")

	// With token bucket and global limiter
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithAlgorithm(TokenBucket))

	set.Wait(context.TODO(), "ut-method")

	// With token bucket and limiter by method
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithAlgorithm(TokenBucket),
		WithReqPerSecByPath("ut-method", 100))

	set.Wait(context.TODO(), "ut-method")

	// With leaky bucket and global limiter
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithAlgorithm(LeakyBucket))

	set.Wait(context.TODO(), "ut-method")

	// With leaky bucket and limiter by method
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithAlgorithm(LeakyBucket),
		WithReqPerSecByPath("ut-method", 100))

	set.Wait(context.TODO(), "ut-method")

	// Without any configuration
	set = newOptionSet(rkgrpcinter.RpcTypeUnaryServer)
	set.Wait(context.TODO(), "ut-method")
}

func assertNotPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, false)
	} else {
		// This should never be called in case of a bug
		assert.True(t, true)
	}
}
