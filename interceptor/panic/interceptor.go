// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcpanic

import (
	"fmt"
	rkmid "github.com/rookie-ninja/rk-entry/middleware"
	rkmidpanic "github.com/rookie-ninja/rk-entry/middleware/panic"
	rkgrpcinter "github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

// UnaryServerInterceptor Create new unary server interceptor.
func UnaryServerInterceptor(opts ...rkmidpanic.Option) grpc.UnaryServerInterceptor {
	set := rkmidpanic.NewOptionSet(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		//rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcMethodKey, info.FullMethod)

		defer func() {
			if recv := recover(); recv != nil {
				var sts *status.Status

				if se, ok := recv.(interface{ GRPCStatus() *status.Status }); ok {
					sts = se.GRPCStatus()
				} else {
					sts = status.New(codes.Internal, fmt.Sprintf("%v", recv))
				}
				err = sts.Err()

				rkgrpcctx.GetEvent(ctx).SetCounter("panic", 1)
				rkgrpcctx.GetLogger(ctx).Error(fmt.Sprintf("panic occurs:\n%s", string(debug.Stack())), zap.Error(err))
			}
		}()

		return handler(ctx, req)
	}
}

// StreamServerInterceptor Create new stream server interceptor.
func StreamServerInterceptor(opts ...rkmidpanic.Option) grpc.StreamServerInterceptor {
	set := rkmidpanic.NewOptionSet(opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		//rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcMethodKey, info.FullMethod)

		defer func() {
			if recv := recover(); recv != nil {
				var sts *status.Status

				if se, ok := recv.(interface{ GRPCStatus() *status.Status }); ok {
					sts = se.GRPCStatus()
				} else {
					sts = status.New(codes.Internal, fmt.Sprintf("%v", recv))
				}
				err = sts.Err()

				rkgrpcctx.GetEvent(stream.Context()).SetCounter("panic", 1)
				rkgrpcctx.GetLogger(stream.Context()).Error(fmt.Sprintf("panic occurs:\n%s", string(debug.Stack())), zap.Error(err))
			}
		}()

		return handler(srv, stream)
	}
}
