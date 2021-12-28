// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcjwt

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
	"testing"
)

func TestNewOptionSet(t *testing.T) {
	// without options
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer)
	assert.NotEmpty(t, set.EntryName)
	assert.NotEmpty(t, set.EntryType)
	assert.False(t, set.Skipper("ut-method"))
	assert.Empty(t, set.SigningKeys)
	assert.Nil(t, set.SigningKey)
	assert.Equal(t, set.SigningAlgorithm, AlgorithmHS256)
	assert.NotNil(t, set.Claims)
	assert.Equal(t, set.TokenLookup, "header:"+headerAuthorization)
	assert.Equal(t, set.AuthScheme, "Bearer")
	assert.Equal(t, reflect.ValueOf(set.KeyFunc).Pointer(), reflect.ValueOf(set.defaultKeyFunc).Pointer())
	assert.Equal(t, reflect.ValueOf(set.ParseTokenFunc).Pointer(), reflect.ValueOf(set.defaultParseToken).Pointer())

	// with options
	skipper := func(string) bool {
		return false
	}
	claims := &fakeClaims{}
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	}
	parseToken := func(string, context.Context) (*jwt.Token, error) { return nil, nil }
	tokenLookups := strings.Join([]string{
		"query:ut-query",
		"param:ut-param",
		"cookie:ut-cookie",
		"form:ut-form",
		"header:ut-header",
	}, ",")

	set = newOptionSet(rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"),
		WithSkipper(skipper),
		WithSigningKey("ut-signing-key"),
		WithSigningKeys("ut-key", "ut-value"),
		WithSigningAlgorithm("ut-signing-algorithm"),
		WithClaims(claims),
		WithTokenLookup(tokenLookups),
		WithAuthScheme("ut-auth-scheme"),
		WithKeyFunc(keyFunc),
		WithParseTokenFunc(parseToken),
		WithIgnorePrefix("/ut"))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
	assert.False(t, set.Skipper("ut-method"))
	assert.Equal(t, "ut-signing-key", set.SigningKey)
	assert.NotEmpty(t, set.SigningKeys)
	assert.Equal(t, "ut-signing-algorithm", set.SigningAlgorithm)
	assert.Equal(t, claims, set.Claims)
	assert.Equal(t, tokenLookups, set.TokenLookup)
	assert.Len(t, set.extractors, 1)
	assert.Equal(t, "ut-auth-scheme", set.AuthScheme)
	assert.Equal(t, reflect.ValueOf(set.KeyFunc).Pointer(), reflect.ValueOf(keyFunc).Pointer())
	assert.Equal(t, reflect.ValueOf(set.ParseTokenFunc).Pointer(), reflect.ValueOf(parseToken).Pointer())
}

type fakeClaims struct{}

func (c *fakeClaims) Valid() error {
	return nil
}
