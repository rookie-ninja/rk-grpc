// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpctokenauth

import (
	"context"
	"encoding/base64"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	inter := UnaryServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"),
		WithToken("token", false))

	assert.NotNil(t, inter)

	set := optionsMap[rkgrpcctx.ToOptionsKey("ut-entry-name", rkgrpcctx.RpcTypeUnaryServer)]
	assert.NotNil(t, set)
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	inter := StreamServerInterceptor(
		WithEntryNameAndType("ut-entry-name", "ut-entry"),
		WithToken("token", false))

	assert.NotNil(t, inter)

	set := optionsMap[rkgrpcctx.ToOptionsKey("ut-entry-name", rkgrpcctx.RpcTypeStreamServer)]
	assert.NotNil(t, set)
}

func TestServerBefore_WithMissingAuthHeader(t *testing.T) {
	ctx := context.TODO()
	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		tokens:    make(map[string]bool),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_WithMissingBearer(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "invalid"))

	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		tokens:    make(map[string]bool),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_WithInvalidBase64(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer invalid"))
	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		tokens:    make(map[string]bool),
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_WithExpiredToken(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("token"))

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer "+encoded))
	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		tokens: map[string]bool{
			"token": true,
		},
	}

	err := serverBefore(ctx, set)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unauthenticated")
}

func TestServerBefore_HappyCase(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("token"))

	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("authorization", "Bearer "+encoded))
	set := &optionSet{
		EntryName: "ut-entry-name",
		EntryType: "ut-entry",
		tokens: map[string]bool{
			"token": false,
		},
	}

	err := serverBefore(ctx, set)
	assert.Nil(t, err)
}
