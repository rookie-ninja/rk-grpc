// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcauth

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, "ut-entry-name", set.EntryName)
	assert.Equal(t, "ut-entry", set.EntryType)
	assert.Equal(t, set,
		optionsMap[rkgrpcinter.ToOptionsKey("ut-entry-name", rkgrpcinter.RpcTypeUnaryServer)])
}
