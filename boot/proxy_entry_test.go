// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	rkentry "github.com/rookie-ninja/rk-entry/entry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
)

type MockServerTransportStream struct {
	method string
}

func (m MockServerTransportStream) Method() string {
	return m.method
}

func (m MockServerTransportStream) SetHeader(md metadata.MD) error {
	panic("implement me")
}

func (m MockServerTransportStream) SendHeader(md metadata.MD) error {
	panic("implement me")
}

func (m MockServerTransportStream) SetTrailer(md metadata.MD) error {
	panic("implement me")
}

func TestNewRule(t *testing.T) {
	// without options
	r := NewRule()
	assert.Empty(t, r.IpPattern)
	assert.Empty(t, r.PathPattern)
	assert.Empty(t, r.HeaderPattern)
	assert.NotNil(t, r.rand)

	// with options
	r = NewRule(
		WithHeaderPatterns(&HeaderPattern{}),
		WithPathPatterns(&PathPattern{}),
		WithIpPatterns(&IpPattern{}))
	assert.NotEmpty(t, r.HeaderPattern)
	assert.NotEmpty(t, r.PathPattern)
	assert.NotEmpty(t, r.IpPattern)
}

func TestRule_MathIpPattern(t *testing.T) {
	ipPattern := &IpPattern{
		Cidrs: []string{"192.168.0.1/24"},
		Dest:  []string{"0.0.0.0"},
	}

	r := NewRule(WithIpPatterns(ipPattern))

	// match IP
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("x-forwarded-remote-addr", "192.168.0.1:1949"))
	matched, dest := r.matchIpPattern(ctx)
	assert.True(t, matched)
	assert.Equal(t, ipPattern.Dest[0], dest)

	// failed to match IP
	ctx = metadata.NewIncomingContext(context.TODO(), metadata.Pairs("x-forwarded-remote-addr", "10.0.0.1:1949"))
	matched, dest = r.matchIpPattern(ctx)
	assert.False(t, matched)
	assert.Empty(t, dest)

	// invalid CIDR
	invalidIpPattern := &IpPattern{
		Cidrs: []string{"invalid"},
		Dest:  []string{"0.0.0.0"},
	}
	r = NewRule(WithIpPatterns(invalidIpPattern))
	ctx = metadata.NewIncomingContext(context.TODO(), metadata.Pairs("x-forwarded-remote-addr", "192.168.0.1:1949"))
	matched, dest = r.matchIpPattern(ctx)
	assert.False(t, matched)
	assert.Empty(t, dest)
}

func TestRule_MatchPathPattern(t *testing.T) {
	pathPattern := &PathPattern{
		Paths: []string{"ut-path"},
		Dest:  []string{"0.0.0.0"},
	}

	r := NewRule(WithPathPatterns(pathPattern))

	// match path
	ctx := grpc.NewContextWithServerTransportStream(context.TODO(), &MockServerTransportStream{
		method: "ut-path",
	})
	matched, dest := r.matchPathPattern(ctx)
	assert.True(t, matched)
	assert.Equal(t, pathPattern.Dest[0], dest)

	// failed to match path
	ctx = grpc.NewContextWithServerTransportStream(context.TODO(), &MockServerTransportStream{
		method: "not-matched",
	})
	matched, dest = r.matchPathPattern(ctx)
	assert.False(t, matched)
	assert.Empty(t, dest)
}

func TestRule_MatchHeaderPattern(t *testing.T) {
	headerPatter := &HeaderPattern{
		Headers: map[string]string{
			"key-1": "val-1",
			"key-2": "val-2",
		},
		Dest: []string{"0.0.0.0"},
	}

	r := NewRule(WithHeaderPatterns(headerPatter))

	// without metadata
	matched, dest := r.matchHeaderPattern(context.TODO())
	assert.False(t, matched)
	assert.Empty(t, dest)

	// match header
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("key-1", "val-1", "key-2", "val-2", "key-3", "val-3"))
	matched, dest = r.matchHeaderPattern(ctx)
	assert.True(t, matched)
	assert.Equal(t, headerPatter.Dest[0], dest)

	// failed to match header
	ctx = metadata.NewIncomingContext(context.TODO(), metadata.Pairs("key-1", "val-1"))
	matched, dest = r.matchHeaderPattern(ctx)
	assert.False(t, matched)
	assert.Empty(t, dest)
}

func TestRule_GetDirector(t *testing.T) {
	// match ip pattern
	ipPattern := &IpPattern{
		Cidrs: []string{"192.168.0.1/24"},
		Dest:  []string{"0.0.0.0"},
	}

	r := NewRule(WithIpPatterns(ipPattern))

	// match IP
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("x-forwarded-remote-addr", "192.168.0.1:1949"))
	ctx, conn, err := r.GetDirector()(ctx)
	assert.NotNil(t, ctx)
	assert.NotNil(t, conn)
	assert.Nil(t, err)

	// failed to match IP, match path
	pathPattern := &PathPattern{
		Paths: []string{"ut-path"},
		Dest:  []string{"0.0.0.0"},
	}
	r = NewRule(WithPathPatterns(pathPattern))
	ctx = grpc.NewContextWithServerTransportStream(context.TODO(), &MockServerTransportStream{
		method: "ut-path",
	})
	ctx, conn, err = r.GetDirector()(ctx)
	assert.NotNil(t, ctx)
	assert.NotNil(t, conn)
	assert.Nil(t, err)

	// failed to math IP, path
	headerPatter := &HeaderPattern{
		Headers: map[string]string{
			"key-1": "val-1",
			"key-2": "val-2",
		},
		Dest: []string{"0.0.0.0"},
	}
	r = NewRule(WithHeaderPatterns(headerPatter))
	ctx = metadata.NewIncomingContext(context.TODO(), metadata.Pairs("key-1", "val-1", "key-2", "val-2", "key-3", "val-3"))
	ctx, conn, err = r.GetDirector()(ctx)
	assert.NotNil(t, ctx)
	assert.NotNil(t, conn)
	assert.Nil(t, err)

	// failed to match any
	r = NewRule()
	ctx, conn, err = r.GetDirector()(ctx)
	assert.Nil(t, ctx)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestNewProxyEntry(t *testing.T) {
	// without entry
	entry := NewProxyEntry()
	assert.Equal(t, ProxyEntryNameDefault, entry.GetName())
	assert.Equal(t, ProxyEntryType, entry.GetType())
	assert.Equal(t, ProxyEntryDescription, entry.GetDescription())
	assert.NotNil(t, entry.EventLoggerEntry)
	assert.NotNil(t, entry.ZapLoggerEntry)

	// with options
	name := "ut-name"
	zapLogger := rkentry.NoopZapLoggerEntry()
	eventLogger := rkentry.NoopEventLoggerEntry()
	rule := &rule{}

	entry = NewProxyEntry(
		WithNameProxy(name),
		WithZapLoggerEntryProxy(zapLogger),
		WithEventLoggerEntryProxy(eventLogger),
		WithRuleProxy(rule))
	assert.Equal(t, name, entry.EntryName)
	assert.Equal(t, zapLogger, entry.ZapLoggerEntry)
	assert.Equal(t, eventLogger, entry.EventLoggerEntry)
	assert.Equal(t, rule, entry.r)
}

func TestProxyEntry_Bootstrap(t *testing.T) {
	defer assertNotPanic(t)

	// without event id in context
	entry := NewProxyEntry()
	entry.Bootstrap(context.TODO())
}

func TestProxyEntry_Interrupt(t *testing.T) {
	defer assertNotPanic(t)

	// without event id in context
	entry := NewProxyEntry()
	entry.Interrupt(context.TODO())
}

func TestProxyEntry_String(t *testing.T) {
	entry := NewProxyEntry()

	assert.NotEmpty(t, entry.String())
}

func TestCodec(t *testing.T) {
	assert.NotNil(t, Codec())
}

func TestCodec_ReadYourWrites(t *testing.T) {
	framePtr := &frame{}
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	codec := rawCodec{}
	require.NoError(t, codec.Unmarshal(data, framePtr), "unmarshalling must go ok")
	out, err := codec.Marshal(framePtr)
	require.NoError(t, err, "no marshal error")
	require.Equal(t, data, out, "output and data must be the same")

	// reuse
	require.NoError(t, codec.Unmarshal([]byte{0x55}, framePtr), "unmarshalling must go ok")
	out, err = codec.Marshal(framePtr)
	require.NoError(t, err, "no marshal error")
	require.Equal(t, []byte{0x55}, out, "output and data must be the same")

}
