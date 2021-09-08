// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcauth

import (
	"encoding/base64"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
}

func TestWithBasicAuth(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithBasicAuth("ut-realm", "user:pass"))

	assert.True(t, set.BasicAccounts[base64.StdEncoding.EncodeToString([]byte("user:pass"))])
}

func TestWithApiKeyAuth(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithApiKeyAuth("ut-api-key"))

	assert.True(t, set.ApiKey["ut-api-key"])
}

func TestWithIgnorePrefix(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithIgnorePrefix("ut-prefix"))

	assert.Contains(t, set.IgnorePrefix, "/ut-prefix")
}

func TestOptionSet_Authorized(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"))

	// With invalid auth type
	assert.False(t, set.Authorized("invalid", ""))

	// With invalid basic auth
	assert.False(t, set.Authorized(typeBasic, "invalid"))
	// With valid basic auth
	assert.True(t, set.Authorized(typeBasic, base64.StdEncoding.EncodeToString([]byte("user:pass"))))

	// With invalid api key
	assert.False(t, set.Authorized(typeApiKey, "invalid"))
	// With valid api key
	assert.True(t, set.Authorized(typeApiKey, "ut-api-key"))
}

func TestOptionSet_ShouldAuth(t *testing.T) {
	// With empty basic auth and api key auth
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"))

	assert.False(t, set.ShouldAuth("ut-method"))

	// With ignoring path
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"),
		WithIgnorePrefix("ut-path"))

	assert.False(t, set.ShouldAuth("/ut-path"))

	// Expect true
	set = newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithBasicAuth("ut-realm", "user:pass"),
		WithApiKeyAuth("ut-api-key"),
		WithIgnorePrefix("ut-path"))
	assert.True(t, set.ShouldAuth("/should-be-true"))
}
