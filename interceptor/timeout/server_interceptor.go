// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpctimeout

import (
	"context"
	rkmid "github.com/rookie-ninja/rk-entry/middleware"
	rkmidtimeout "github.com/rookie-ninja/rk-entry/middleware/timeout"
	rkgrpcerr "github.com/rookie-ninja/rk-grpc/boot/error"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
)

var defaultResponse = rkgrpcerr.Canceled("Request timed out!").Err()

// UnaryServerInterceptor Add timeout interceptors.
func UnaryServerInterceptor(opts ...rkmidtimeout.Option) grpc.UnaryServerInterceptor {
	set := rkmidtimeout.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(nil, rkgrpcctx.GetEvent(ctx))
		toCtx := &unaryTimeoutCtx{
			req:     req,
			grpcCtx: ctx,
			handler: handler,
			before:  beforeCtx,
		}
		// assign handlers
		beforeCtx.Input.InitHandler = unaryInitHandler(toCtx)
		beforeCtx.Input.NextHandler = unaryNextHandler(toCtx)
		beforeCtx.Input.PanicHandler = unaryPanicHandler(toCtx)
		beforeCtx.Input.FinishHandler = unaryFinishHandler(toCtx)
		beforeCtx.Input.TimeoutHandler = unaryTimeoutHandler(toCtx)
		// call before
		set.Before(beforeCtx)

		beforeCtx.Output.WaitFunc()

		return toCtx.resp, toCtx.err
	}
}

// StreamServerInterceptor Add rate limit interceptors.
func StreamServerInterceptor(opts ...rkmidtimeout.Option) grpc.StreamServerInterceptor {
	set := rkmidtimeout.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

		beforeCtx := set.BeforeCtx(nil, rkgrpcctx.GetEvent(wrappedStream.WrappedContext))
		toCtx := &streamTimeoutCtx{
			srv:     srv,
			stream:  stream,
			handler: handler,
			before:  beforeCtx,
		}
		// assign handlers
		beforeCtx.Input.InitHandler = streamInitHandler(toCtx)
		beforeCtx.Input.NextHandler = streamNextHandler(toCtx)
		beforeCtx.Input.PanicHandler = streamPanicHandler(toCtx)
		beforeCtx.Input.FinishHandler = streamFinishHandler(toCtx)
		beforeCtx.Input.TimeoutHandler = streamTimeoutHandler(toCtx)
		// call before
		set.Before(beforeCtx)

		beforeCtx.Output.WaitFunc()

		return toCtx.err
	}
}

// *************** utility ***************

type unaryTimeoutCtx struct {
	req     interface{}
	resp    interface{}
	err     error
	grpcCtx context.Context
	handler grpc.UnaryHandler
	before  *rkmidtimeout.BeforeCtx
}

func unaryTimeoutHandler(ctx *unaryTimeoutCtx) func() {
	return func() {
		ctx.err = defaultResponse
	}
}

func unaryFinishHandler(ctx *unaryTimeoutCtx) func() {
	return func() {}
}

func unaryPanicHandler(ctx *unaryTimeoutCtx) func() {
	return func() {}
}

func unaryNextHandler(ctx *unaryTimeoutCtx) func() {
	return func() {
		ctx.resp, ctx.err = ctx.handler(ctx.grpcCtx, ctx.req)
	}
}

func unaryInitHandler(ctx *unaryTimeoutCtx) func() {
	return func() {}
}

type streamTimeoutCtx struct {
	srv     interface{}
	stream  grpc.ServerStream
	err     error
	handler grpc.StreamHandler
	before  *rkmidtimeout.BeforeCtx
}

func streamTimeoutHandler(ctx *streamTimeoutCtx) func() {
	return func() {
		ctx.err = defaultResponse
	}
}

func streamFinishHandler(ctx *streamTimeoutCtx) func() {
	return func() {}
}

func streamPanicHandler(ctx *streamTimeoutCtx) func() {
	return func() {}
}

func streamNextHandler(ctx *streamTimeoutCtx) func() {
	return func() {
		ctx.err = ctx.handler(ctx.srv, ctx.stream)
	}
}

func streamInitHandler(ctx *streamTimeoutCtx) func() {
	return func() {}
}
