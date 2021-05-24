// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcpanic

import (
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		defer func() {
			if err := recover(); err != nil {
				event := rkgrpcctx.GetEvent(ctx)
				rkgrpcctx.GetZapLogger(ctx).Error("panic occurs\n"+string(debug.Stack()),
					zap.Any("err", err))
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(
							strings.ToLower(se.Error()),
							"broken pipe") || strings.Contains(strings.ToLower(se.Error()),
							"connection reset by peer") {
							brokenPipe = true
							event.AddErr(se)
						}
					}
				}

				if brokenPipe {
					rkgrpcctx.GetZapLogger(ctx).Error(string(debug.Stack()))
					event.SetEndTime(time.Now())
					event.SetResCode(codes.Internal.String())
					return
				}

				event.SetEndTime(time.Now())
				event.SetResCode(codes.Internal.String())
			}
		}()

		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		defer func() {
			if err := recover(); err != nil {
				event := rkgrpcctx.GetEvent(ctx)
				rkgrpcctx.GetZapLogger(ctx).Error("panic occurs\n"+string(debug.Stack()),
					zap.Any("err", err))
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(
							strings.ToLower(se.Error()),
							"broken pipe") || strings.Contains(strings.ToLower(se.Error()),
							"connection reset by peer") {
							brokenPipe = true
							event.AddErr(se)
						}
					}
				}

				if brokenPipe {
					rkgrpcctx.GetZapLogger(ctx).Error(string(debug.Stack()))
					event.SetEndTime(time.Now())
					event.SetResCode(codes.Internal.String())
					return
				}

				event.SetEndTime(time.Now())
				event.SetResCode(codes.Internal.String())
			}
		}()

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if err := recover(); err != nil {
				event := rkgrpcctx.GetEvent(ctx)
				rkgrpcctx.GetZapLogger(ctx).Error("panic occurs\n"+string(debug.Stack()),
					zap.Any("err", err))
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(
							strings.ToLower(se.Error()),
							"broken pipe") || strings.Contains(strings.ToLower(se.Error()),
							"connection reset by peer") {
							brokenPipe = true
							event.AddErr(se)
						}
					}
				}

				if brokenPipe {
					rkgrpcctx.GetZapLogger(ctx).Error(string(debug.Stack()))
					event.SetEndTime(time.Now())
					event.SetResCode(codes.Internal.String())
					return
				}

				event.SetEndTime(time.Now())
				event.SetResCode(codes.Internal.String())
			}
		}()

		return handler(ctx, req)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			wrappedStream := rkgrpcctx.WrapServerStream(stream)
			ctx := wrappedStream.Context()

			if err := recover(); err != nil {
				event := rkgrpcctx.GetEvent(ctx)
				rkgrpcctx.GetZapLogger(ctx).Error("panic occurs\n" + string(debug.Stack()))
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(
							strings.ToLower(se.Error()),
							"broken pipe") || strings.Contains(strings.ToLower(se.Error()),
							"connection reset by peer") {
							brokenPipe = true
							event.AddErr(se)
						}
					}
				}

				if brokenPipe {
					rkgrpcctx.GetZapLogger(ctx).Error(string(debug.Stack()))
					event.SetEndTime(time.Now())
					event.SetResCode(codes.Internal.String())
					return
				}

				event.SetEndTime(time.Now())
				event.SetResCode(codes.Internal.String())
			}
		}()

		return handler(srv, stream)
	}
}
