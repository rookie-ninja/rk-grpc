// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpctimeout

import (
	"context"
	"fmt"
	rkmidtimeout "github.com/rookie-ninja/rk-entry/middleware/timeout"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

func TestUnaryServerInterceptor_WithTimeout(t *testing.T) {
	// with global timeout
	inter := UnaryServerInterceptor(rkmidtimeout.WithTimeout(time.Nanosecond))

	resp, err := inter(context.TODO(), req, unaryInfo, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, defaultResponse, err)

	// with method
	inter = UnaryServerInterceptor(
		rkmidtimeout.WithTimeoutByPath(unaryInfo.FullMethod, time.Nanosecond))

	resp, err = inter(context.TODO(), req, unaryInfo, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, defaultResponse, err)
}

func TestUnaryServerInterceptor_WithPanic(t *testing.T) {
	defer assertPanic(t)

	inter := UnaryServerInterceptor(
		rkmidtimeout.WithTimeout(time.Second))

	resp, err := inter(context.TODO(), req, unaryInfo, panicHandlerUnary)
	assert.Nil(t, resp)
	assert.Nil(t, err)
}

func TestUnaryServerInterceptor_HappyCase(t *testing.T) {
	timeoutMethod := "/timeout"
	happyMethod := "/happy"

	// Let's add two routes /timeout and /happy
	// We expect interceptor acts as the name describes
	inter := UnaryServerInterceptor(
		rkmidtimeout.WithTimeoutByPath(timeoutMethod, time.Nanosecond),
		rkmidtimeout.WithTimeoutByPath(happyMethod, time.Second))

	// timeout on /timeout
	resp, err := inter(context.TODO(), req, &grpc.UnaryServerInfo{FullMethod: timeoutMethod}, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, defaultResponse, err)

	// OK on /happy
	resp, err = inter(context.TODO(), req, &grpc.UnaryServerInfo{FullMethod: happyMethod}, returnHandlerUnary)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
}

func TestStreamServerInterceptor_WithTimeout(t *testing.T) {
	// with global timeout
	inter := StreamServerInterceptor(
		rkmidtimeout.WithTimeout(time.Nanosecond))

	err := inter(fakeServer, stream, streamInfo, sleepHandlerStream)
	assert.Equal(t, defaultResponse, err)

	// with method
	inter = StreamServerInterceptor(
		rkmidtimeout.WithTimeoutByPath(streamInfo.FullMethod, time.Nanosecond))

	err = inter(fakeServer, stream, streamInfo, sleepHandlerStream)
	assert.Equal(t, defaultResponse, err)
}

func TestStreamServerInterceptor_WithPanic(t *testing.T) {
	defer assertPanic(t)

	inter := StreamServerInterceptor(
		rkmidtimeout.WithTimeout(time.Second))

	err := inter(fakeServer, stream, streamInfo, panicHandlerStream)
	assert.Nil(t, err)
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	timeoutMethod := "/timeout"
	happyMethod := "/happy"

	// Let's add two routes /timeout and /happy
	// We expect interceptor acts as the name describes
	inter := StreamServerInterceptor(
		rkmidtimeout.WithTimeoutByPath(timeoutMethod, time.Nanosecond),
		rkmidtimeout.WithTimeoutByPath(happyMethod, time.Second))

	// timeout on /timeout
	err := inter(fakeServer, stream, &grpc.StreamServerInfo{
		FullMethod: timeoutMethod,
	}, sleepHandlerStream)
	assert.Equal(t, defaultResponse, err)

	// OK on /happy
	err = inter(fakeServer, stream, &grpc.StreamServerInfo{
		FullMethod: happyMethod,
	}, returnHandlerStream)
	assert.Nil(t, err)
}

// ************ Test utility ************

var (
	unaryInfo = &grpc.UnaryServerInfo{
		FullMethod: "/ut-method",
	}

	streamInfo = &grpc.StreamServerInfo{
		FullMethod: "/ut-method",
	}

	stream = FakeServerStream{
		ctx: context.TODO(),
	}

	fakeServer = &FakeServer{}

	req = "fake-request"
)

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

func sleepHandlerUnary(ctx context.Context, req interface{}) (interface{}, error) {
	time.Sleep(time.Second)
	return nil, nil
}

func sleepHandlerStream(srv interface{}, stream grpc.ServerStream) error {
	time.Sleep(time.Second)
	return nil
}

func panicHandlerUnary(ctx context.Context, req interface{}) (interface{}, error) {
	panic(fmt.Errorf("ut panic"))
}

func panicHandlerStream(srv interface{}, stream grpc.ServerStream) error {
	panic(fmt.Errorf("ut panic"))
}

func returnHandlerUnary(ctx context.Context, req interface{}) (interface{}, error) {
	return "ut-resp", nil
}

func returnHandlerStream(srv interface{}, stream grpc.ServerStream) error {
	return nil
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
