// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcctx

import (
	"context"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"testing"
)

func TestErrorToCodesFuncDefault_HappyCase(t *testing.T) {
	assert.Equal(t, codes.OK, ErrorToCodesFuncDefault(nil))
}

func TestContainsPayload_WithNilContext(t *testing.T) {
	assert.False(t, ContainsPayload(nil))
}

func TestContainsPayload_ExpectTrue(t *testing.T) {
	ctx := context.WithValue(context.TODO(), key, &payload{})
	assert.True(t, ContainsPayload(ctx))
}

func TestContainsPayload_ExpectFalse(t *testing.T) {
	assert.False(t, ContainsPayload(context.TODO()))
}

func TestWithEvent_HappyCase(t *testing.T) {
	event := rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEvent()

	opt := WithEvent(event)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Equal(t, event, GetEvent(ctx))
}

func TestWithEvent_WithNilInput(t *testing.T) {
	opt := WithEvent(nil)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.NotNil(t, GetEvent(ctx))
}

func TestWithZapLogger_HappyCase(t *testing.T) {
	logger := rklogger.NoopLogger

	opt := WithZapLogger(logger)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Equal(t, logger, GetZapLogger(ctx))
}

func TestWithZapLogger_WithNilInput(t *testing.T) {
	opt := WithZapLogger(nil)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.NotNil(t, GetZapLogger(ctx))
}

func TestWithEntryName_HappyCase(t *testing.T) {
	opt := WithEntryName("ut-entry")
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Equal(t, "ut-entry", GetEntryName(ctx))
}

func TestWithIncomingMD_HappyCase(t *testing.T) {
	meta := metadata.Pairs()

	opt := WithIncomingMD(meta)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Equal(t, meta, GetIncomingMD(ctx))
}

func TestWithIncomingMD_WithNilInput(t *testing.T) {
	opt := WithIncomingMD(nil)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.NotNil(t, GetIncomingMD(ctx))
}

func TestWithOutgoingMD_HappyCase(t *testing.T) {
	meta := metadata.Pairs()

	opt := WithOutgoingMD(meta)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Equal(t, meta, GetOutgoingMD(ctx))
}

func TestWithOutgoingMD_WithNilInput(t *testing.T) {
	opt := WithOutgoingMD(nil)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.NotNil(t, GetOutgoingMD(ctx))
}

func TestWithWithRpcInfo_HappyCase(t *testing.T) {
	info := &RpcInfo{}

	opt := WithRpcInfo(info)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Equal(t, info, GetRpcInfo(ctx))
}

func TestWithRpcInfo_WithNilInput(t *testing.T) {
	opt := WithRpcInfo(nil)
	ctx := ContextWithPayload(context.TODO(), opt)
	assert.NotNil(t, ctx)
	assert.Nil(t, GetRpcInfo(ctx))
}

func TestContextWithPayload_WithNilInput(t *testing.T) {
	ctx := ContextWithPayload(nil)
	assert.NotNil(t, ctx)
	assert.NotNil(t, GetEvent(ctx))
	assert.NotNil(t, GetZapLogger(ctx))
	assert.NotEmpty(t, GetEntryName(ctx))
	assert.NotNil(t, GetIncomingMD(ctx))
	assert.NotNil(t, GetOutgoingMD(ctx))
	assert.NotNil(t, GetPayload(ctx))
	assert.NotNil(t, getPayloadRaw(ctx))
}

func TestContextWithPayload_HappyCase(t *testing.T) {
	event := rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEvent()
	logger := rklogger.NoopLogger
	entryName := "ut-entry-name"
	incomingMD := metadata.Pairs()
	outgoingMD := metadata.Pairs()
	info := &RpcInfo{}

	ctx := ContextWithPayload(context.TODO(),
		WithEvent(event),
		WithZapLogger(logger),
		WithEntryName(entryName),
		WithIncomingMD(incomingMD),
		WithOutgoingMD(outgoingMD),
		WithRpcInfo(info))

	assert.NotNil(t, ctx)
	assert.Equal(t, event, GetEvent(ctx))
	assert.Equal(t, logger, GetZapLogger(ctx))
	assert.NotEmpty(t, GetEntryName(ctx))
	assert.Equal(t, incomingMD, GetIncomingMD(ctx))
	assert.Equal(t, outgoingMD, GetOutgoingMD(ctx))
	assert.NotNil(t, GetPayload(ctx))
	assert.NotNil(t, getPayloadRaw(ctx))
}

func TestNewContext_HappyCase(t *testing.T) {
	ctx := NewContext()
	assert.NotNil(t, ctx)
	assert.NotNil(t, GetEvent(ctx))
	assert.NotNil(t, GetZapLogger(ctx))
	assert.NotEmpty(t, GetEntryName(ctx))
	assert.NotNil(t, GetIncomingMD(ctx))
	assert.NotNil(t, GetOutgoingMD(ctx))
	assert.NotNil(t, GetPayload(ctx))
	assert.NotNil(t, getPayloadRaw(ctx))
}

func TestAddToOutgoingMD_WithNil(t *testing.T) {
	assertNotPanic(t)
	AddToOutgoingMD(nil, "key", "value")
}

func TestAddToOutgoingMD_WithoutMD(t *testing.T) {
	assertNotPanic(t)
	ctx := context.TODO()
	AddToOutgoingMD(ctx, "key", "value")

	md, ok := metadata.FromOutgoingContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, md)
}

func TestAddToOutgoingMD_WithMD(t *testing.T) {
	assertNotPanic(t)
	out := metadata.Pairs()
	ctx := metadata.NewOutgoingContext(context.TODO(), out)

	AddToOutgoingMD(ctx, "key", "value")

	md, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.NotNil(t, md)
	assert.Empty(t, md["key"])
}

func TestAddToOutgoingMD_HappyCase(t *testing.T) {
	assertNotPanic(t)
	ctx := NewContext()
	AddToOutgoingMD(ctx, "key", "value")

	md := GetOutgoingMD(ctx)
	assert.NotNil(t, md)
	assert.Len(t, md["key"], 1)
	assert.Equal(t, "value", md["key"][0])
}

func TestAddRequestIdToOutgoingMD_HappyCase(t *testing.T) {
	assertNotPanic(t)
	ctx := NewContext()
	AddRequestIdToOutgoingMD(ctx)

	md := GetOutgoingMD(ctx)
	assert.NotNil(t, md)
	assert.Len(t, md[RequestIdKeyDefault], 1)
	assert.NotEmpty(t, md[RequestIdKeyDefault][0])
}

func TestGetEvent_WithNilContext(t *testing.T) {
	assertNotPanic(t)
	event := GetEvent(nil)
	assert.NotNil(t, event)
}

func TestGetEvent_WithoutRkContext(t *testing.T) {
	assertNotPanic(t)
	event := GetEvent(context.TODO())
	assert.NotNil(t, event)
}

func TestGetEvent_HappyCase(t *testing.T) {
	ctx := NewContext()
	event := rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEvent()
	GetPayload(ctx).event = event

	assert.Equal(t, event, GetEvent(ctx))
}

func TestGetZapLogger_WithNilContext(t *testing.T) {
	assertNotPanic(t)
	logger := GetZapLogger(nil)
	assert.NotNil(t, logger)
}

func TestGetZapLogger_WithoutRkContext(t *testing.T) {
	assertNotPanic(t)
	logger := GetZapLogger(context.TODO())
	assert.NotNil(t, logger)
}

func TestGetZapLogger_HappyCase(t *testing.T) {
	ctx := NewContext()
	logger := rklogger.NoopLogger
	GetPayload(ctx).zapLogger = logger

	assert.Equal(t, logger, GetZapLogger(ctx))
}

func TestSetZapLogger_HappyCase(t *testing.T) {
	ctx := NewContext()
	logger := rklogger.NoopLogger

	SetZapLogger(ctx, logger)

	assert.Equal(t, logger, GetZapLogger(ctx))
}

func TestGetIncomingMD_WithNil(t *testing.T) {
	md := GetIncomingMD(nil)
	assert.NotNil(t, md)
}

func TestGetIncomingMD_WithoutRkContext(t *testing.T) {
	originMD := metadata.Pairs()
	ctx := metadata.NewIncomingContext(context.TODO(), originMD)

	outMD := GetIncomingMD(ctx)
	assert.NotNil(t, outMD)
	assert.Equal(t, originMD, outMD)
}

func TestGetIncomingMD_HappyCase(t *testing.T) {
	ctx := NewContext()

	assert.NotNil(t, GetIncomingMD(ctx))
}

func TestGetOutgoingMD_WithNil(t *testing.T) {
	md := GetOutgoingMD(nil)
	assert.NotNil(t, md)
}

func TestOutgoingMD_WithoutRkContext(t *testing.T) {
	originMD := metadata.Pairs()
	ctx := metadata.NewOutgoingContext(context.TODO(), originMD)

	outMD := GetOutgoingMD(ctx)
	assert.NotNil(t, outMD)
	assert.Equal(t, originMD, outMD)
}

func TestGetOutgoingMD_HappyCase(t *testing.T) {
	ctx := NewContext()

	assert.NotNil(t, GetOutgoingMD(ctx))
}

func TestGetValueFromIncomingMD_WithNilContext(t *testing.T) {
	res := GetValueFromIncomingMD(nil, "ut-key")
	assert.Empty(t, res)
}

func TestGetValueFromIncomingMD_WithoutRkContext(t *testing.T) {
	res := GetValueFromIncomingMD(context.TODO(), "ut-key")
	assert.Empty(t, res)
}

func TestGetValueFromIncomingMD_HappyCase(t *testing.T) {
	ctx := NewContext()
	GetPayload(ctx).incomingMD.Set("ut-key", "ut-value")

	res := GetValueFromIncomingMD(ctx, "ut-key")
	assert.Len(t, res, 1)
	assert.Equal(t, res[0], "ut-value")
}

func TestGetValueFromOutgoingMD_WithoutRkContext(t *testing.T) {
	res := GetValueFromOutgoingMD(context.TODO(), "ut-key")
	assert.Empty(t, res)
}

func TestGetValueFromOutgoingMD_HappyCase(t *testing.T) {
	ctx := NewContext()
	GetPayload(ctx).outgoingMD.Set("ut-key", "ut-value")

	res := GetValueFromOutgoingMD(ctx, "ut-key")
	assert.Len(t, res, 1)
	assert.Equal(t, res[0], "ut-value")
}

func TestGetRequestIdsFromOutgoingMD_WithNilContext(t *testing.T) {
	ids := GetRequestIdsFromOutgoingMD(nil)
	assert.Empty(t, ids)
}

func TestGetRequestIdsFromOutgoingMD_WithoutRkContext(t *testing.T) {
	md := metadata.Pairs(
		RequestIdKeyDash, "dash",
		RequestIdKeyLowerCase, "lower",
		RequestIdKeyUnderline, "underline")
	ctx := metadata.NewOutgoingContext(context.TODO(), md)

	ids := GetRequestIdsFromOutgoingMD(ctx)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "dash")
	assert.Contains(t, ids, "lower")
	assert.Contains(t, ids, "underline")
}

func TestGetRequestIdsFromOutgoingMD_HappyCase(t *testing.T) {
	ctx := NewContext()
	AddToOutgoingMD(ctx, RequestIdKeyDash, "dash")
	AddToOutgoingMD(ctx, RequestIdKeyLowerCase, "lower")
	AddToOutgoingMD(ctx, RequestIdKeyUnderline, "underline")

	ids := GetRequestIdsFromOutgoingMD(ctx)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "dash")
	assert.Contains(t, ids, "lower")
	assert.Contains(t, ids, "underline")
}

func TestGetRequestIdsFromIncomingMD_WithNilContext(t *testing.T) {
	ids := GetRequestIdsFromIncomingMD(nil)
	assert.Empty(t, ids)
}

func TestGetRequestIdsFromIncomingMD_WithoutRkContext(t *testing.T) {
	md := metadata.Pairs(
		RequestIdKeyDash, "dash",
		RequestIdKeyLowerCase, "lower",
		RequestIdKeyUnderline, "underline")
	ctx := metadata.NewIncomingContext(context.TODO(), md)

	ids := GetRequestIdsFromIncomingMD(ctx)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "dash")
	assert.Contains(t, ids, "lower")
	assert.Contains(t, ids, "underline")
}

func TestGetRequestIdsFromIncomingMD_HappyCase(t *testing.T) {
	ctx := NewContext()
	GetPayload(ctx).incomingMD.Set(RequestIdKeyDash, "dash")
	GetPayload(ctx).incomingMD.Set(RequestIdKeyLowerCase, "lower")
	GetPayload(ctx).incomingMD.Set(RequestIdKeyUnderline, "underline")

	ids := GetRequestIdsFromIncomingMD(ctx)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "dash")
	assert.Contains(t, ids, "lower")
	assert.Contains(t, ids, "underline")
}

func TestGenerateRequestId_HappyCase(t *testing.T) {
	assert.NotEmpty(t, GenerateRequestId())
}

func TestGenerateRequestIdWithPrefix_HappyCase(t *testing.T) {
	id := GenerateRequestIdWithPrefix("ut-prefix")
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "ut-prefix")
}

func TestGetPayload_WithNilContext(t *testing.T) {
	payload := GetPayload(nil)
	assert.NotNil(t, payload)
}

func TestGetPayload_HappyCase(t *testing.T) {
	payload := &payload{}
	ctx := context.WithValue(NewContext(), key, payload)

	res := GetPayload(ctx)
	assert.Equal(t, payload, res)
}

func TestGetEntryName_WithNilContext(t *testing.T) {
	res := GetEntryName(nil)
	assert.Equal(t, NewFakeGRPCEntry().GetName(), res)
}

func TestGetEntryName_WithoutRkContext(t *testing.T) {
	ctx := context.TODO()
	res := GetEntryName(ctx)
	assert.Equal(t, NewFakeGRPCEntry().GetName(), res)
}

func TestGetEntryName_HappyCase(t *testing.T) {
	ctx := ContextWithPayload(context.TODO(), WithEntryName("ut-entry"))
	res := GetEntryName(ctx)
	assert.Equal(t, "ut-entry", res)
}

func TestGetRpcInfo_WithNilContext(t *testing.T) {
	assert.Nil(t, GetRpcInfo(nil))
}

func TestGetRpcInfo_WithoutRkContext(t *testing.T) {
	ctx := context.TODO()
	assert.Nil(t, GetRpcInfo(ctx))
}

func TestGetRpcInfo_HappyCase(t *testing.T) {
	info := &RpcInfo{}
	ctx := ContextWithPayload(
		context.TODO(),
		WithRpcInfo(info))
	assert.Equal(t, info, GetRpcInfo(ctx))
}

func assertPanic(t *testing.T) {
	if r := recover(); r != nil {
		// expect panic to be called with non nil error
		assert.True(t, true)
	} else {
		// this should never be called in case of a bug
		assert.True(t, false)
	}
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
