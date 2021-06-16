// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasicauth

import (
	"context"
	"encoding/base64"
	rkgrpcbasic "github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"),
		WithCredential("user:pass"))

	assert.NotNil(t, inter)

	set := optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"),
		WithCredential("user:pass"))

	assert.NotNil(t, inter)

	set := optionsMap[rkgrpcbasic.ToOptionsKey("ut-entry-name", rkgrpcbasic.RpcTypeStreamServer)]
	assert.NotNil(t, set)
}

func TestServerBefore_WithMissingAuthHeader(t *testing.T) {
	ctx := context.TODO()
	set := &optionSet{
		EntryName:   "ut-entry-name",
		EntryType:   "ut-entry",
		credentials: make(map[string]string),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_WithMissingBearer(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "invalid"))

	set := &optionSet{
		EntryName:   "ut-entry-name",
		EntryType:   "ut-entry",
		credentials: make(map[string]string),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_WithInvalidBase64(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer invalid"))
	set := &optionSet{
		EntryName:   "ut-entry-name",
		EntryType:   "ut-entry",
		credentials: make(map[string]string),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_WithInvalidCred(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("pass"))

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer "+encoded))
	set := &optionSet{
		EntryName:   "ut-entry-name",
		EntryType:   "ut-entry",
		credentials: make(map[string]string),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_InvalidPass(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("user:invalid-pass"))

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer "+encoded))
	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		credentials: map[string]string{
			"user": "pass",
		},
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_HappyCase(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("user:pass"))

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer "+encoded))
	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		credentials: map[string]string{
			"user": "pass",
		},
	}

	err := serverBefore(ctx, set)
	assert.Nil(t, err)
}
