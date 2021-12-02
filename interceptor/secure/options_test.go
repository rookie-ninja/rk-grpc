// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcsec

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewOptionSet(t *testing.T) {
	// without options
	set := newOptionSet()
	assert.NotEmpty(t, set.EntryName)
	assert.NotEmpty(t, set.EntryType)
	assert.NotNil(t, set.Skipper)
	assert.Equal(t, "1; mode=block", set.XSSProtection)
	assert.Equal(t, "nosniff", set.ContentTypeNosniff)
	assert.Equal(t, "SAMEORIGIN", set.XFrameOptions)
	assert.Zero(t, set.HSTSMaxAge)
	assert.False(t, set.HSTSExcludeSubdomains)
	assert.False(t, set.HSTSPreloadEnabled)
	assert.Empty(t, set.CSPReportOnly)
	assert.Empty(t, set.ReferrerPolicy)
	assert.Empty(t, set.IgnorePrefix)

	// with options
	set = newOptionSet(
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithSkipper(func(r *http.Request) bool {
			return true
		}),
		WithXSSProtection("ut-xss"),
		WithContentTypeNosniff("ut-sniff"),
		WithXFrameOptions("ut-frame"),
		WithHSTSMaxAge(10),
		WithHSTSExcludeSubdomains(true),
		WithHSTSPreloadEnabled(true),
		WithContentSecurityPolicy("ut-policy"),
		WithCSPReportOnly(true),
		WithReferrerPolicy("ut-ref"),
		WithIgnorePrefix("ut-prefix"),
	)
	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
	assert.True(t, set.Skipper(nil))
	assert.Equal(t, "ut-xss", set.XSSProtection)
	assert.Equal(t, "ut-sniff", set.ContentTypeNosniff)
	assert.Equal(t, "ut-frame", set.XFrameOptions)
	assert.Equal(t, 10, set.HSTSMaxAge)
	assert.True(t, set.HSTSExcludeSubdomains)
	assert.True(t, set.HSTSPreloadEnabled)
	assert.Equal(t, "ut-policy", set.ContentSecurityPolicy)
	assert.True(t, set.CSPReportOnly)
	assert.Equal(t, "ut-ref", set.ReferrerPolicy)
	assert.NotEmpty(t, set.IgnorePrefix)
}
