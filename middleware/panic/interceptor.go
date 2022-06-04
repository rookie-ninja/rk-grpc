// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcpanic

import (
	"fmt"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/panic"
	"github.com/rookie-ninja/rk-grpc/v2/middleware"
	"github.com/rookie-ninja/rk-grpc/v2/middleware/context"
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
		ctx = rkgrpcmid.WrapContextForServer(ctx)
		rkgrpcmid.AddToServerContextPayload(ctx, rkmid.EntryNameKey, set.GetEntryName())

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
		wrappedStream.WrappedContext = rkgrpcmid.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcmid.AddToServerContextPayload(wrappedStream.WrappedContext, rkmid.EntryNameKey, set.GetEntryName())

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
