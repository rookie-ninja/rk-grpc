// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasicauth

import (
	rkgrpcbasic "github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithEntryNameAndType_HappyCase(t *testing.T) {
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry-name", "ut-entry"))

	assert.Equal(t, "ut-entry-name", set.EntryName)
	assert.Equal(t, "ut-entry", set.EntryType)
	assert.Equal(t, set,
		optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeUnaryServer)])
}

func TestWithCredential_HappyCase(t *testing.T) {
	credOne := "user-1:pass-1"
	credTwo := "user-2:pass-2"

	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer,
		WithCredential(credOne, credTwo))

	assert.Equal(t, "pass-1", set.credentials["user-1"])
	assert.Equal(t, "pass-2", set.credentials["user-2"])
}

func TestOptionSet_Authorized_HappyCase(t *testing.T) {
	cred := "ut-user:ut-pass"

	set := &optionSet{
		credentials: make(map[string]string),
	}
	opt := WithCredential(cred)
	opt(set)

	assert.True(t, set.Authorized("ut-user", "ut-pass"))
	assert.False(t, set.Authorized("ut-user", "fake-pass"))
	assert.False(t, set.Authorized("fake-user", "fake-pass"))
}
