// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpctimeout

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"time"
)

// UnaryServerInterceptor Add rate limit interceptors.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		event := rkgrpcctx.GetEvent(ctx)

		rk := set.getTimeoutRk(info.FullMethod)

		// 1: create three channels
		//
		// finishChan: triggered while request has been handled successfully
		// panicChan: triggered while panic occurs
		// timeoutChan: triggered while timing out
		finishChan := make(chan struct{}, 1)
		panicChan := make(chan interface{}, 1)
		timeoutChan := time.After(rk.timeout)

		var resp interface{}
		var err error

		// 2: create a new go routine catch panic
		go func() {
			defer func() {
				if recv := recover(); recv != nil {
					panicChan <- recv
				}
			}()

			resp, err = handler(ctx, req)
			finishChan <- struct{}{}
		}()

		// 3: waiting for three channels
		select {
		// 3.1: return panic
		case recv := <-panicChan:
			panic(recv)
		// 3.2: break
		case <-finishChan:
			break
		// 3.3: return timeout error
		case <-timeoutChan:
			// set as timeout
			event.SetCounter("timeout", 1)
			err = rk.response
		}

		return resp, err
	}
}

// StreamServerInterceptor Add rate limit interceptors.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		event := rkgrpcctx.GetEvent(wrappedStream.Context())

		rk := set.getTimeoutRk(info.FullMethod)

		// 1: create three channels
		//
		// finishChan: triggered while request has been handled successfully
		// panicChan: triggered while panic occurs
		// timeoutChan: triggered while timing out
		finishChan := make(chan struct{}, 1)
		panicChan := make(chan interface{}, 1)
		timeoutChan := time.After(rk.timeout)

		var err error

		// 2: create a new go routine catch panic
		go func() {
			defer func() {
				if recv := recover(); recv != nil {
					panicChan <- recv
				}
			}()

			err = handler(srv, wrappedStream)
			finishChan <- struct{}{}
		}()

		// 3: waiting for three channels
		select {
		// 3.1: return panic
		case recv := <-panicChan:
			panic(recv)
		// 3.2: break
		case <-finishChan:
			break
		// 3.3: return timeout error
		case <-timeoutChan:
			// set as timeout
			event.SetCounter("timeout", 1)
			err = rk.response
		}

		return err
	}
}
