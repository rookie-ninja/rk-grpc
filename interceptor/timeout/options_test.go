// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpctimeout

import (
	"github.com/rookie-ninja/rk-grpc/boot/error"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

var utResp = rkgrpcerr.Canceled("Request timed out!").Err()

func TestWithEntryNameAndType(t *testing.T) {
	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithEntryNameAndType("ut-entry", "ut-type"))

	assert.Equal(t, "ut-entry", set.EntryName)
	assert.Equal(t, "ut-type", set.EntryType)
}

func TestWithTimeoutAndResp(t *testing.T) {
	// happy case
	opt := WithTimeoutAndResp(time.Second, utResp)
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opt)
	assert.Equal(t, time.Second, set.getTimeoutRk(global).timeout)
	assert.Equal(t, reflect.ValueOf(utResp).Pointer(), reflect.ValueOf(set.getTimeoutRk(global).response).Pointer())

	// with nil response
	opt = WithTimeoutAndResp(time.Second, nil)
	set = newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opt)
	assert.Equal(t, time.Second, set.getTimeoutRk(global).timeout)
	assert.Equal(t, reflect.ValueOf(defaultResponse).Pointer(), reflect.ValueOf(set.getTimeoutRk(global).response).Pointer())
}

func TestWithTimeoutAndRespByPath(t *testing.T) {
	p := "/ut-path"

	// happy case
	opt := WithTimeoutAndRespByPath(p, time.Second, utResp)
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opt)
	assert.Equal(t, time.Second, set.getTimeoutRk(p).timeout)
	assert.Equal(t, reflect.ValueOf(utResp).Pointer(), reflect.ValueOf(set.getTimeoutRk(p).response).Pointer())

	// with nil response
	opt = WithTimeoutAndRespByPath(p, time.Second, nil)
	set = newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opt)
	assert.Equal(t, time.Second, set.getTimeoutRk(p).timeout)
	assert.Equal(t, reflect.ValueOf(defaultResponse).Pointer(), reflect.ValueOf(set.getTimeoutRk(p).response).Pointer())
}

func TestNormalisePath(t *testing.T) {
	withoutSlash := "ut-path"
	withSlash := "/ut-path"
	// without slash
	assert.Equal(t, withSlash, normalisePath(withoutSlash))
	assert.Equal(t, withSlash, normalisePath(withSlash))
}

func TestGetTimeoutRk(t *testing.T) {
	// with path
	p := "/ut-path"

	set := newOptionSet(
		rkgrpcinter.RpcTypeUnaryServer,
		WithTimeoutAndRespByPath(p, time.Second, nil))
	timeout := set.getTimeoutRk(p)
	assert.NotNil(t, timeout)
	assert.Equal(t, timeout, set.timeouts[p])

	// without path
	timeout = set.getTimeoutRk("/invalid-path")
	assert.NotNil(t, timeout)
	assert.Equal(t, timeout, set.timeouts[global])
}
