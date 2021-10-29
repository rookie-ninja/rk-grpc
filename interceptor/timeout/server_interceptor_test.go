// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpctimeout

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

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

func TestUnaryServerInterceptor_WithoutOptions(t *testing.T) {
	inter := UnaryServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeUnaryServer)])
}

func TestUnaryServerInterceptor_WithTimeout(t *testing.T) {
	// with global timeout
	inter := UnaryServerInterceptor(
		WithTimeoutAndResp(time.Nanosecond, nil))

	resp, err := inter(context.TODO(), req, unaryInfo, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, defaultResponse, err)

	// with method
	inter = UnaryServerInterceptor(
		WithTimeoutAndRespByPath(unaryInfo.FullMethod, time.Nanosecond, nil))

	resp, err = inter(context.TODO(), req, unaryInfo, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, defaultResponse, err)

	// with custom err
	myErr := fmt.Errorf("my error")
	inter = UnaryServerInterceptor(
		WithTimeoutAndRespByPath(unaryInfo.FullMethod, time.Nanosecond, myErr))

	resp, err = inter(context.TODO(), req, unaryInfo, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, myErr, err)
}

func TestUnaryServerInterceptor_WithPanic(t *testing.T) {
	defer assertPanic(t)

	inter := UnaryServerInterceptor(
		WithTimeoutAndResp(time.Second, nil))

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
		WithTimeoutAndRespByPath(timeoutMethod, time.Nanosecond, nil),
		WithTimeoutAndRespByPath(happyMethod, time.Second, nil))

	// timeout on /timeout
	resp, err := inter(context.TODO(), req, &grpc.UnaryServerInfo{FullMethod: timeoutMethod}, sleepHandlerUnary)
	assert.Nil(t, resp)
	assert.Equal(t, defaultResponse, err)

	// OK on /happy
	resp, err = inter(context.TODO(), req, &grpc.UnaryServerInfo{FullMethod: happyMethod}, returnHandlerUnary)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
}

func TestStreamServerInterceptor_WithoutOptions(t *testing.T) {
	inter := StreamServerInterceptor()

	assert.NotNil(t, inter)
	assert.NotNil(t, optionsMap[rkgrpcinter.ToOptionsKey(rkgrpcinter.RpcEntryNameValue, rkgrpcinter.RpcTypeStreamServer)])
}

func TestStreamServerInterceptor_WithTimeout(t *testing.T) {
	// with global timeout
	inter := StreamServerInterceptor(
		WithTimeoutAndResp(time.Nanosecond, nil))

	err := inter(fakeServer, stream, streamInfo, sleepHandlerStream)
	assert.Equal(t, defaultResponse, err)

	// with method
	inter = StreamServerInterceptor(
		WithTimeoutAndRespByPath(streamInfo.FullMethod, time.Nanosecond, nil))

	err = inter(fakeServer, stream, streamInfo, sleepHandlerStream)
	assert.Equal(t, defaultResponse, err)

	// with custom err
	myErr := fmt.Errorf("my error")
	inter = StreamServerInterceptor(
		WithTimeoutAndRespByPath(streamInfo.FullMethod, time.Nanosecond, myErr))

	err = inter(fakeServer, stream, streamInfo, sleepHandlerStream)
	assert.Equal(t, myErr, err)
}

func TestStreamServerInterceptor_WithPanic(t *testing.T) {
	defer assertPanic(t)

	inter := StreamServerInterceptor(
		WithTimeoutAndResp(time.Second, nil))

	err := inter(fakeServer, stream, streamInfo, panicHandlerStream)
	assert.Nil(t, err)
}

func TestStreamServerInterceptor_HappyCase(t *testing.T) {
	timeoutMethod := "/timeout"
	happyMethod := "/happy"

	// Let's add two routes /timeout and /happy
	// We expect interceptor acts as the name describes
	inter := StreamServerInterceptor(
		WithTimeoutAndRespByPath(timeoutMethod, time.Nanosecond, nil),
		WithTimeoutAndRespByPath(happyMethod, time.Second, nil))

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

func assertPanic(t *testing.T) {
	if r := recover(); r != nil {
		// expect panic to be called with non nil error
		assert.True(t, true)
	} else {
		// this should never be called in case of a bug
		assert.True(t, false)
	}
}
