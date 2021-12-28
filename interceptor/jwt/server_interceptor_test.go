// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcjwt

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
	"testing"
)

var (
	fakeRequest  = &FakeRequest{}
	fakeResponse = &FakeResponse{}
	fakeServer   = &FakeServer{}
)

type FakeRequest struct{}

type FakeResponse struct{}

type FakeServer struct{}

type FakeServerStream struct {
	ctx context.Context
}

func (f FakeServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (f FakeServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (f FakeServerStream) SetTrailer(md metadata.MD) {
	return
}

func (f FakeServerStream) Context() context.Context {
	return f.ctx
}

func (f FakeServerStream) SendMsg(m interface{}) error {
	return nil
}

func (f FakeServerStream) RecvMsg(m interface{}) error {
	return nil
}

func TestUnaryServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	// with skipper
	inter := UnaryServerInterceptor(
		WithSkipper(func(string) bool {
			return true
		}))
	info := &grpc.UnaryServerInfo{
		FullMethod: "/ut-method",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}
	res, err := inter(context.TODO(), fakeRequest, info, handler)
	assert.NotNil(t, res)
	assert.Nil(t, err)

	// without options
	inter = UnaryServerInterceptor()
	info = &grpc.UnaryServerInfo{
		FullMethod: "/ut-method",
	}
	handler = func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}
	res, err = inter(context.TODO(), fakeRequest, info, handler)
	assert.Nil(t, res)
	assert.NotNil(t, err)

	// with parse token error
	parseTokenErrFunc := func(auth string, c context.Context) (*jwt.Token, error) {
		return nil, errors.New("ut-error")
	}
	inter = UnaryServerInterceptor(WithParseTokenFunc(parseTokenErrFunc))
	info = &grpc.UnaryServerInfo{
		FullMethod: "/ut-method",
	}
	handler = func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}
	ctx := context.WithValue(context.TODO(), headerAuthorization, strings.Join([]string{"Bearer", "ut-auth"}, " "))
	res, err = inter(ctx, fakeRequest, info, handler)
	assert.Nil(t, res)
	assert.NotNil(t, err)

	// happy case
	parseTokenErrFunc = func(auth string, c context.Context) (*jwt.Token, error) {
		return &jwt.Token{}, nil
	}
	inter = UnaryServerInterceptor(WithParseTokenFunc(parseTokenErrFunc))
	info = &grpc.UnaryServerInfo{
		FullMethod: "/ut-method",
	}
	handler = func(ctx context.Context, req interface{}) (interface{}, error) {
		return FakeResponse{}, nil
	}
	ctx = context.WithValue(context.TODO(), headerAuthorization, strings.Join([]string{"Bearer", "ut-auth"}, " "))
	res, err = inter(ctx, fakeRequest, info, handler)
	assert.Nil(t, res)
	assert.NotNil(t, err)
}

func TestStreamServerInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	// with skipper
	inter := StreamServerInterceptor(
		WithSkipper(func(string) bool {
			return true
		}))
	info := &grpc.StreamServerInfo{
		FullMethod: "/ut-method",
	}
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	err := inter(fakeServer, &FakeServerStream{
		ctx: context.TODO(),
	}, info, handler)
	assert.Nil(t, err)

	// without options
	inter = StreamServerInterceptor()
	info = &grpc.StreamServerInfo{
		FullMethod: "/ut-method",
	}
	handler = func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	err = inter(fakeServer, &FakeServerStream{
		ctx: context.TODO(),
	}, info, handler)
	assert.NotNil(t, err)

	// with parse token error
	parseTokenErrFunc := func(auth string, c context.Context) (*jwt.Token, error) {
		return nil, errors.New("ut-error")
	}
	inter = StreamServerInterceptor(WithParseTokenFunc(parseTokenErrFunc))
	info = &grpc.StreamServerInfo{
		FullMethod: "/ut-method",
	}
	handler = func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	ctx := context.WithValue(context.TODO(), headerAuthorization, strings.Join([]string{"Bearer", "ut-auth"}, " "))
	err = inter(fakeServer, &FakeServerStream{
		ctx: ctx,
	}, info, handler)
	assert.NotNil(t, err)

	// happy case
	parseTokenErrFunc = func(auth string, c context.Context) (*jwt.Token, error) {
		return &jwt.Token{}, nil
	}
	inter = StreamServerInterceptor(WithParseTokenFunc(parseTokenErrFunc))
	info = &grpc.StreamServerInfo{
		FullMethod: "/ut-method",
	}
	handler = func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}
	ctx = context.WithValue(context.TODO(), headerAuthorization, strings.Join([]string{"Bearer", "ut-auth"}, " "))
	err = inter(fakeServer, &FakeServerStream{
		ctx: ctx,
	}, info, handler)
	assert.NotNil(t, err)
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
