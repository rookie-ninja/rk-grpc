// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcmid

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type FakeAddr struct{}

func (f FakeAddr) Network() string {
	return "ut-net"
}

func (f FakeAddr) String() string {
	return "0.0.0.0:0"
}

func TestGetGwInfo(t *testing.T) {
	md := metadata.New(map[string]string{
		"x-forwarded-method":     "ut-method",
		"x-forwarded-scheme":     "ut-scheme",
		"x-forwarded-user-agent": "ut-agent",
		"x-forwarded-pattern":    "ut-path/{id}",
	})

	method, path, scheme, agent := GetGwInfo(md)
	assert.Equal(t, "ut-method", method)
	assert.Equal(t, "ut-path/{id}", path)
	assert.Equal(t, "ut-scheme", scheme)
	assert.Equal(t, "ut-agent", agent)
}

func TestGetGrpcInfo(t *testing.T) {
	fullMethod := "/service/method"

	service, method := GetGrpcInfo(fullMethod)
	assert.Equal(t, "method", method)
	assert.Equal(t, "service", service)
}

func TestToOptionsKey(t *testing.T) {
	entryName, rpcType := "ut-entry", "ut-rpc"
	assert.Equal(t, "ut-entry-ut-rpc", ToOptionsKey(entryName, rpcType))
}

func TestGetRemoteAddressSetFromMeta(t *testing.T) {
	md := metadata.New(map[string]string{
		"x-forwarded-remote-addr": "ip:port",
	})

	ip, port := GetRemoteAddressSetFromMeta(md)
	assert.Equal(t, "ip", ip)
	assert.Equal(t, "port", port)

	// With case of ::1
	md = metadata.New(map[string]string{
		"x-forwarded-remote-addr": "[::1]:port",
	})

	ip, port = GetRemoteAddressSetFromMeta(md)
	assert.Equal(t, "localhost", ip)
	assert.Equal(t, "port", port)
}

func TestGetRemoteAddressSet(t *testing.T) {
	// Happy case
	md := metadata.New(map[string]string{
		"x-forwarded-remote-addr": "0.0.0.0:0",
	})

	ctx := metadata.NewOutgoingContext(context.TODO(), md)
	ip, port, netType := GetRemoteAddressSet(ctx)
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, "0", port)
	assert.Empty(t, netType)

	// Not in context, inject into peer
	ctx = peer.NewContext(context.TODO(), &peer.Peer{
		Addr: &FakeAddr{},
	})
	ip, port, netType = GetRemoteAddressSet(ctx)
	assert.Equal(t, "0.0.0.0", ip)
	assert.Equal(t, "0", port)
	assert.Equal(t, "ut-net", netType)

	// with x-forwarded-for header
	ctx = peer.NewContext(context.TODO(), &peer.Peer{
		Addr: &FakeAddr{},
	})
	ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
		"x-forwarded-for": "::1",
	}))

	ip, port, netType = GetRemoteAddressSet(ctx)
	assert.Equal(t, "localhost", ip)
	assert.Equal(t, "0", port)
	assert.Equal(t, "ut-net", netType)
}

func TestMergeToOutgoingMD(t *testing.T) {
	// Without existing outgoing MD
	md := metadata.New(map[string]string{
		"key": "value",
	})
	ctx := MergeToOutgoingMD(context.TODO(), md)
	md2, _ := metadata.FromOutgoingContext(ctx)
	assert.Equal(t, md, md2)

	// With appended already
	md3 := metadata.New(map[string]string{
		"key3": "value3",
	})
	ctx = MergeToOutgoingMD(ctx, md3)
	md4, _ := metadata.FromOutgoingContext(ctx)
	assert.Len(t, md4, 2)
}

func TestMergeAndDeduplicateSlice(t *testing.T) {
	src := []string{
		"a", "b", "c",
	}
	target := []string{
		"a", "d",
	}

	res := MergeAndDeduplicateSlice(src, target)
	assert.Len(t, res, 4)
}

func TestWrapContextForServer(t *testing.T) {
	// For the first time
	ctx := WrapContextForServer(context.TODO())
	assert.True(t, ContainsServerPayload(ctx))

	// Wrap it again
	assert.Equal(t, ctx, WrapContextForServer(ctx))
}

func TestGetServerContextPayload(t *testing.T) {
	// With nil context
	assert.NotNil(t, GetServerContextPayload(nil))

	// Without payload in server context
	assert.NotNil(t, GetServerContextPayload(nil))

	// With payload in context
	ctx := WrapContextForServer(context.TODO())
	assert.NotNil(t, GetServerContextPayload(ctx))
}

func TestAddToServerContextPayload(t *testing.T) {
	defer assertNotPanic(t)

	// With nil value
	AddToServerContextPayload(context.TODO(), "key", nil)

	// Happy case
	ctx := WrapContextForServer(context.TODO())
	AddToServerContextPayload(ctx, "key", "value")
	assert.Equal(t, "value", GetServerContextPayload(ctx)["key"])
}

func TestContainsServerPayload(t *testing.T) {
	// Expect true
	ctx := WrapContextForServer(context.TODO())
	assert.True(t, ContainsServerPayload(ctx))

	// Expect false
	assert.False(t, ContainsServerPayload(context.TODO()))
}

func TestGetServerPayloadKey(t *testing.T) {
	assert.NotNil(t, GetServerPayloadKey())
}

func assertNotPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, false)
	} else {
		// This should never be called in case of a bug
		assert.True(t, true)
	}
}
