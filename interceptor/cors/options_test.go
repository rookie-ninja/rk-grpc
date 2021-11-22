// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccors

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewOptionSet(t *testing.T) {
	// With empty option
	set := newOptionSet()
	assert.Equal(t, rkgrpcinter.RpcEntryNameValue, set.EntryName)
	assert.Equal(t, rkgrpcinter.RpcEntryTypeValue, set.EntryType)
	assert.NotNil(t, set.Skipper)
	assert.NotEmpty(t, set.AllowOrigins)
	assert.NotEmpty(t, set.AllowMethods)
	assert.NotEmpty(t, set.allowPatterns)
	assert.Empty(t, set.AllowHeaders)
	assert.Empty(t, set.ExposeHeaders)
	assert.False(t, set.AllowCredentials)
	assert.Zero(t, set.MaxAge)

	// With options
	set = newOptionSet(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithSkipper(func(*http.Request) bool {
			return true
		}),
		WithAllowOrigins("ut-origin"),
		WithAllowMethods(http.MethodPost),
		WithAllowHeaders("ut-header"),
		WithAllowCredentials(true),
		WithExposeHeaders("ut-header"),
		WithMaxAge(1))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
	assert.NotNil(t, set.Skipper)
	assert.NotEmpty(t, set.AllowOrigins)
	assert.NotEmpty(t, set.AllowMethods)
	assert.NotEmpty(t, set.allowPatterns)
	assert.NotEmpty(t, set.AllowHeaders)
	assert.NotEmpty(t, set.ExposeHeaders)
	assert.True(t, set.AllowCredentials)
	assert.Equal(t, 1, set.MaxAge)
}

func TestIsOriginAllowed(t *testing.T) {
	set := newOptionSet()

	// 1: wildcard
	set.AllowOrigins = []string{"*"}
	set.toPatterns()
	assert.True(t, set.isOriginAllowed("http://ut.domain"))

	// 2: exact matching
	set.AllowOrigins = []string{"http://ut.domain"}
	set.toPatterns()
	assert.True(t, set.isOriginAllowed("http://ut.domain"))
	assert.False(t, set.isOriginAllowed("http://ut.another"))

	// 3: subdomain
	set.AllowOrigins = []string{"http://*.ut.domain"}
	set.toPatterns()
	assert.True(t, set.isOriginAllowed("http://sub.ut.domain"))
	assert.True(t, set.isOriginAllowed("http://sub.sub.ut.domain"))
	assert.False(t, set.isOriginAllowed("http://ut.domain"))
	assert.False(t, set.isOriginAllowed("http://ut.another"))

	// 4: wildcard in middle of domain
	set.AllowOrigins = []string{"http://ut.*.domain"}
	set.toPatterns()
	assert.True(t, set.isOriginAllowed("http://ut.sub.domain"))
	assert.True(t, set.isOriginAllowed("http://ut.sub.sub.domain"))
	assert.False(t, set.isOriginAllowed("http://ut.domain"))
	assert.False(t, set.isOriginAllowed("http://ut.another"))

	// 5: wildcard in the last
	set.AllowOrigins = []string{"http://ut.domain.*"}
	set.toPatterns()
	assert.True(t, set.isOriginAllowed("http://ut.domain.sub"))
	assert.True(t, set.isOriginAllowed("http://ut.domain.sub.sub"))
	assert.False(t, set.isOriginAllowed("http://ut.domain"))
	assert.False(t, set.isOriginAllowed("http://ut.another"))
}
