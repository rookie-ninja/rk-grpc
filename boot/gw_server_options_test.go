// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	"errors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	testhttp "github.com/stretchr/testify/http"
	"google.golang.org/grpc/metadata"
	"io"
	"net/http"
	"testing"
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
	assert.Equal(t, "ut", key)
}
