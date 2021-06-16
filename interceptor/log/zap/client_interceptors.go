// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"path"
	"time"
)

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 1: Before invoking
		ctx = clientBefore(ctx, set)

		// Set headers for internal usage
		md := rkgrpcctx.GetIncomingMD(ctx)
		opts = append(opts, grpc.Header(&md))

		// 2: Invoking
		err := invoker(ctx, method, req, resp, cc, opts...)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// 3: After invoking
		clientAfter(ctx, set)

		return err
	}
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Before invoking
		ctx = clientBefore(ctx, set)

		// Set headers for internal usage
		md := rkgrpcctx.GetIncomingMD(ctx)
		opts = append(opts, grpc.Header(&md))

		// Invoking
		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// After invoking
		clientAfter(ctx, set)

		return clientStream, err
	}
}

func clientBefore(ctx context.Context, set *optionSet) context.Context {
	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)

	event := set.EventLoggerEntry.GetEventFactory().CreateEvent()
	event.SetStartTime(time.Now())

	event.SetRemoteAddr(rpcInfo.RemoteIp + ":" + rpcInfo.RemotePort)
	event.SetOperation(path.Base(rpcInfo.GrpcMethod))

	payloads := []zap.Field{
		zap.String("remoteIp", rpcInfo.RemoteIp),
		zap.String("remotePort", rpcInfo.RemotePort),
		zap.String("grpcService", path.Dir(rpcInfo.GrpcMethod)[1:]),
		zap.String("grpcMethod", path.Base(rpcInfo.GrpcMethod)),
		zap.String("grpcType", rpcInfo.Type),
	}

	logger := set.ZapLoggerEntry.GetLogger()

	if d, ok := ctx.Deadline(); ok {
		payloads = append(payloads, zap.String("deadline", d.Format(time.RFC3339)))
	}

	event.AddPayloads(payloads...)

	// Extract outgoing metadata from context
	outgoingMD := rkgrpcctx.GetOutgoingMD(ctx)
	incomingMD := rkgrpcctx.GetIncomingMD(ctx)

	return rkgrpcctx.ToRkContext(ctx,
		rkgrpcctx.WithEvent(event),
		rkgrpcctx.WithZapLogger(logger),
		rkgrpcctx.WithIncomingMD(incomingMD),
		rkgrpcctx.WithOutgoingMD(outgoingMD),
	)
}

func clientAfter(ctx context.Context, set *optionSet) {
	event := rkgrpcctx.GetEvent(ctx)

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)
	event.AddErr(rpcInfo.Err)
	code := set.ErrorToCodeFunc(rpcInfo.Err)
	endTime := time.Now()

	// Check whether context is cancelled from server
	select {
	case <-ctx.Done():
		event.AddErr(ctx.Err())
	default:
		break
	}

	// Extract request id and log it
	incomingRequestId := rkgrpcctx.GetRequestId(ctx)

	if len(incomingRequestId) > 0 {
		event.SetEventId(incomingRequestId)
		event.SetRequestId(incomingRequestId)
		event.SetTraceId(rkgrpcctx.GetTraceId(ctx))
	}

	event.SetResCode(code.String())
	event.SetEndTime(endTime)
	event.Finish()
}
