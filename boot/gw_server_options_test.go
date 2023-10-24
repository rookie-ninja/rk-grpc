// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	testhttp "github.com/stretchr/testify/http"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

type FakeEncoder struct{}

func (FakeEncoder) Encode(interface{}) error {
	return errors.New("not implemented")
}

type FakeMarshaller struct{}

func (f FakeMarshaller) Marshal(v interface{}) ([]byte, error) {
	return []byte{}, nil
}

func (f FakeMarshaller) Unmarshal(data []byte, v interface{}) error {
	return nil
}

func (f FakeMarshaller) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderWrapper{}
}

func (f FakeMarshaller) NewEncoder(w io.Writer) runtime.Encoder {
	return FakeEncoder{}
}

func (f FakeMarshaller) ContentType(v interface{}) string {
	return ""
}

func TestHttpErrorHandler(t *testing.T) {
	defer assertNotPanic(t)

	md := runtime.ServerMetadata{
		HeaderMD: metadata.New(map[string]string{
			"k1": "v1",
		}),
		TrailerMD: metadata.New(map[string]string{
			"k2": "v2",
		}),
	}
	ctx := runtime.NewServerMetadataContext(context.TODO(), md)
	marshaler := FakeMarshaller{}
	writer := &testhttp.TestResponseWriter{}
	request := &http.Request{
		Header: http.Header{},
	}

	request.Header.Set("TE", "trailers")
	HttpErrorHandler(ctx, nil, marshaler, writer, request, nil)
}

func TestOutgoingHeaderMatcher(t *testing.T) {
	key, ok := OutgoingHeaderMatcher("ut")
	assert.True(t, ok)
	assert.Equal(t, "ut", key)
}

func TestIncomingHeaderMatcher(t *testing.T) {
	key, ok := IncomingHeaderMatcher("ut")
	assert.True(t, ok)
	assert.Equal(t, "Ut", key)

	// forbidden header
	key, ok = IncomingHeaderMatcher("Connection")
	assert.False(t, ok)
	assert.Empty(t, key)
}

func TestToMarshalOptions(t *testing.T) {
	// with nil gwOption
	assert.NotNil(t, toMarshalOptions(nil))

	// with nil gwOption.Marshal
	assert.NotNil(t, toMarshalOptions(&gwOption{}))

	// with all fields in gwOption.Marshal to be nil
	gwOptStr := `
---
marshal:
`
	gwOpt := &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	mOpt := toMarshalOptions(gwOpt)
	assert.NotNil(t, mOpt)
	assert.False(t, mOpt.Multiline)
	assert.False(t, mOpt.EmitUnpopulated)
	assert.Empty(t, mOpt.Indent)
	assert.False(t, mOpt.AllowPartial)
	assert.False(t, mOpt.UseProtoNames)
	assert.False(t, mOpt.UseEnumNumbers)

	// with fields
	gwOptStr = `
---
marshal:
  multiline: true
  emitUnpopulated: true
  indent: "ut-indent"
  allowPartial: true
  useProtoNames: true
  useEnumNumbers: true
`
	gwOpt = &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	mOpt = toMarshalOptions(gwOpt)
	assert.NotNil(t, mOpt)
	assert.True(t, mOpt.Multiline)
	assert.True(t, mOpt.EmitUnpopulated)
	assert.Equal(t, "ut-indent", mOpt.Indent)
	assert.True(t, mOpt.AllowPartial)
	assert.True(t, mOpt.UseProtoNames)
	assert.True(t, mOpt.UseEnumNumbers)
}

func TestToUnmarshalOptions(t *testing.T) {
	// with nil gwOption
	assert.NotNil(t, toUnmarshalOptions(nil))

	// with nil gwOption.Unmarshal
	assert.NotNil(t, toUnmarshalOptions(&gwOption{}))

	// with all fields in gwOption.Marshal to be nil
	gwOptStr := `
---
unmarshal:
`
	gwOpt := &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	uOpt := toUnmarshalOptions(gwOpt)
	assert.NotNil(t, uOpt)
	assert.False(t, uOpt.AllowPartial)
	assert.False(t, uOpt.DiscardUnknown)

	// with fields
	gwOptStr = `
---
unmarshal:
  allowPartial: true
  discardUnknown: true
`
	gwOpt = &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	uOpt = toUnmarshalOptions(gwOpt)
	assert.NotNil(t, uOpt)
	assert.True(t, uOpt.AllowPartial)
	assert.True(t, uOpt.DiscardUnknown)
}

func TestMergeWithRkGwMarshalOption(t *testing.T) {
	// with nil gwOption
	mOpt := mergeWithRkGwMarshalOption(nil)
	assert.NotNil(t, mOpt)
	assert.False(t, mOpt.Multiline)
	assert.True(t, mOpt.EmitUnpopulated)
	assert.Empty(t, mOpt.Indent)
	assert.False(t, mOpt.AllowPartial)
	assert.False(t, mOpt.UseProtoNames)
	assert.False(t, mOpt.UseEnumNumbers)

	// with nil gwOption.Marshal
	mOpt = mergeWithRkGwMarshalOption(&gwOption{})
	assert.NotNil(t, mOpt)
	assert.False(t, mOpt.Multiline)
	assert.True(t, mOpt.EmitUnpopulated)
	assert.Empty(t, mOpt.Indent)
	assert.False(t, mOpt.AllowPartial)
	assert.False(t, mOpt.UseProtoNames)
	assert.False(t, mOpt.UseEnumNumbers)

	// with all fields in gwOption.Marshal to be nil
	gwOptStr := `
---
marshal:
`
	gwOpt := &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	mOpt = mergeWithRkGwMarshalOption(gwOpt)
	assert.NotNil(t, mOpt)
	assert.False(t, mOpt.Multiline)
	assert.True(t, mOpt.EmitUnpopulated)
	assert.Empty(t, mOpt.Indent)
	assert.False(t, mOpt.AllowPartial)
	assert.False(t, mOpt.UseProtoNames)
	assert.False(t, mOpt.UseEnumNumbers)

	// with fields
	gwOptStr = `
---
marshal:
  multiline: true
  emitUnpopulated: true
  indent: "ut-indent"
  allowPartial: true
  useProtoNames: true
  useEnumNumbers: true
`
	gwOpt = &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	mOpt = mergeWithRkGwMarshalOption(gwOpt)
	assert.NotNil(t, mOpt)
	assert.True(t, mOpt.Multiline)
	assert.True(t, mOpt.EmitUnpopulated)
	assert.Equal(t, "ut-indent", mOpt.Indent)
	assert.True(t, mOpt.AllowPartial)
	assert.True(t, mOpt.UseProtoNames)
	assert.True(t, mOpt.UseEnumNumbers)
}

func TestMergeWithRkGwUnmarshalOption(t *testing.T) {
	// with nil gwOption
	uOpt := mergeWithRkGwUnmarshalOption(nil)
	assert.NotNil(t, uOpt)
	assert.False(t, uOpt.AllowPartial)
	assert.False(t, uOpt.DiscardUnknown)

	// with nil gwOption.Marshal
	uOpt = mergeWithRkGwUnmarshalOption(&gwOption{})
	assert.NotNil(t, uOpt)
	assert.False(t, uOpt.AllowPartial)
	assert.False(t, uOpt.DiscardUnknown)

	// with all fields in gwOption.Marshal to be nil
	gwOptStr := `
---
unmarshal:
`
	gwOpt := &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	uOpt = mergeWithRkGwUnmarshalOption(gwOpt)
	assert.NotNil(t, uOpt)
	assert.False(t, uOpt.AllowPartial)
	assert.False(t, uOpt.DiscardUnknown)

	// with fields
	gwOptStr = `
---
unmarshal:
  allowPartial: true
  discardUnknown: true
`
	gwOpt = &gwOption{}
	assert.Nil(t, yaml.Unmarshal([]byte(gwOptStr), gwOpt))
	uOpt = mergeWithRkGwUnmarshalOption(gwOpt)
	assert.NotNil(t, uOpt)
	assert.True(t, uOpt.AllowPartial)
	assert.True(t, uOpt.DiscardUnknown)
}

func TestNewRkGwServerMuxOptions(t *testing.T) {
	// with nil marshal and unmarshal option
	opts := NewRkGwServerMuxOptions(nil, nil)
	assert.NotNil(t, opts)
	assert.Len(t, opts, 5)

	// with marshal and unmarshal option
	mOptIn := &protojson.MarshalOptions{}
	uOptIn := &protojson.UnmarshalOptions{}
	opts = NewRkGwServerMuxOptions(mOptIn, uOptIn)
	assert.NotNil(t, opts)
	assert.Len(t, opts, 5)
}

func TestRkGwMetadataBuilder(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/uc-path", nil)
	ctxAnnotator := runtime.WithHTTPPathPattern("/uc/path/{id}")
	ctx := ctxAnnotator(context.Background())

	md := rkGwMetadataBuilder(ctx, req)
	assert.Equal(t, metadata.Pairs(
		"x-forwarded-method", req.Method,
		"x-forwarded-path", req.URL.Path,
		"x-forwarded-scheme", "http",
		"x-forwarded-remote-addr", req.RemoteAddr,
		"x-forwarded-user-agent", req.UserAgent(),
		"x-forwarded-pattern", "/uc/path/{id}"), md)
}

// ************ Test utility ************

func assertNotPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, false)
	} else {
		// This should never be called in case of a bug
		assert.True(t, true)
	}
}

func assertPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, true)
	} else {
		// This should never be called in case of a bug
		assert.True(t, false)
	}
}
