// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcmeta

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryClient,
		WithEntryNameAndType("ut-entry", "ut-type"))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
	assert.Equal(t, "RK", set.Prefix)
	assert.NotEmpty(t, set.AppNameKey)
	assert.NotEmpty(t, set.AppVersionKey)
	assert.NotEmpty(t, set.AppUnixTimeKey)
	assert.NotEmpty(t, set.ReceivedTimeKey)
}

func TestWithPrefix(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryClient,
		WithPrefix("ut-prefix"))

	assert.Equal(t, "ut-prefix", set.Prefix)
	assert.NotEmpty(t, set.AppNameKey)
	assert.NotEmpty(t, set.AppVersionKey)
	assert.NotEmpty(t, set.AppUnixTimeKey)
	assert.NotEmpty(t, set.ReceivedTimeKey)
}
