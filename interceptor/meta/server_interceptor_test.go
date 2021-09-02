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

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	opt := WithEntryNameAndType("ut-name", "ut-type")

	set := &optionSet{}

	opt(set)

	assert.Equal(t, "ut-name", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
}

func TestWithPrefix_HappyCase(t *testing.T) {
	opt := WithPrefix("ut-prefix")

	set := &optionSet{}

	opt(set)

	assert.Equal(t, "ut-prefix", set.Prefix)
}

func TestExtensionInterceptor_WithoutOption(t *testing.T) {
	UnaryServerInterceptor()

	assert.NotEmpty(t, optionsMap)
}

func TestExtensionInterceptor_HappyCase(t *testing.T) {
	UnaryServerInterceptor(WithEntryNameAndType("ut-name", "ut-type"))

	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey("ut-name", rkgrpcinter.RpcTypeUnaryServer)])
}
