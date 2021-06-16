// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	rkgrpcbasic "github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"path"
	"time"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Before invoking
		ctx = serverBefore(ctx, set)

		// Invoking
		resp, err := handler(ctx, req)

		if rpcInfo := rkgrpcctx.GetRpcInfo(ctx); rpcInfo != nil {
			rpcInfo.Err = err
		}

		// After invoking
		serverAfter(ctx, set)

		return resp, err
	}
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		ctx := serverBefore(wrappedStream.WrappedContext, set)

		// Invoking
		err := handler(srv, wrappedStream)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// After invoking
		serverAfter(ctx, set)

		return err
	}
}

func serverBefore(ctx context.Context, set *optionSet) context.Context {
	event := set.EventLoggerEntry.GetEventFactory().CreateEvent()
	event.SetStartTime(time.Now())

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)

	payloads := []zap.Field{
		zap.String("grpcService", rpcInfo.GrpcService),
		zap.String("grpcMethod", rpcInfo.GrpcMethod),
		zap.String("grpcType", rpcInfo.Type),
		zap.String("gwMethod", rpcInfo.GwMethod),
		zap.String("gwPath", rpcInfo.GwPath),
		zap.String("gwScheme", rpcInfo.GwScheme),
		zap.String("gwUserAgent", rpcInfo.GwUserAgent),
	}

	// handle payloads
	event.AddPayloads(payloads...)

	// handle remote address
	event.SetRemoteAddr(rpcInfo.RemoteIp + ":" + rpcInfo.RemoteIp)

	// handle operation
	event.SetOperation(path.Base(rpcInfo.GrpcMethod))

	if _, ok := ctx.Deadline(); ok {
		event.AddErr(ctx.Err())
	}

	return rkgrpcctx.ToRkContext(ctx,
		rkgrpcctx.WithEvent(event),
		rkgrpcctx.WithZapLogger(set.ZapLoggerEntry.GetLogger()),
	)
}

func serverAfter(ctx context.Context, options *optionSet) {
	event := rkgrpcctx.GetEvent(ctx)

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)
	event.AddErr(rpcInfo.Err)
	code := options.ErrorToCodeFunc(rpcInfo.Err)
	endTime := time.Now()

	// Check whether context is cancelled from client
	select {
	case <-ctx.Done():
		event.AddErr(ctx.Err())
	default:
		break
	}

	event.SetResCode(code.String())
	event.SetEndTime(endTime)
	event.Finish()
}
