// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"path"
	"time"
)

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcctx.RpcTypeUnaryClient, opts...)

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
	set := newOptionSet(rkgrpcctx.RpcTypeStreamClient, opts...)

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

	outgoingRequestIds := rkgrpcctx.GetRequestIdsFromOutgoingMD(ctx)

	remoteIP, remotePort, _ := net.SplitHostPort(rpcInfo.Target)
	event.SetRemoteAddr(remoteIP)
	event.SetOperation(path.Base(rpcInfo.GrpcMethod))

	fields := []zap.Field{
		rkgrpcctx.Realm,
		rkgrpcctx.Region,
		rkgrpcctx.AZ,
		rkgrpcctx.Domain,
		zap.String("appName", rkentry.GlobalAppCtx.GetAppInfoEntry().AppName),
		zap.String("appVersion", rkentry.GlobalAppCtx.GetAppInfoEntry().Version),
		rkgrpcctx.LocalIp,
		zap.String("remoteIp", remoteIP),
		zap.String("remotePort", remotePort),
		zap.String("grpcService", path.Dir(rpcInfo.GrpcMethod)[1:]),
		zap.String("grpcMethod", path.Base(rpcInfo.GrpcMethod)),
		zap.String("grpcType", rpcInfo.Type),
		zap.Strings("outgoingRequestId", outgoingRequestIds),
		zap.Time("startTime", event.GetStartTime()),
	}

	logger := set.ZapLoggerEntry.GetLogger().With(
		zap.Strings("outgoingRequestId", outgoingRequestIds))

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	event.AddFields(fields...)

	// Extract outgoing metadata from context
	outgoingMD := rkgrpcctx.GetOutgoingMD(ctx)
	incomingMD := rkgrpcctx.GetIncomingMD(ctx)

	return rkgrpcctx.ContextWithPayload(ctx,
		rkgrpcctx.WithEvent(event),
		rkgrpcctx.WithZapLogger(logger),
		rkgrpcctx.WithIncomingMD(incomingMD),
		rkgrpcctx.WithOutgoingMD(outgoingMD),
	)
}

func clientAfter(ctx context.Context, set *optionSet) {
	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)
	code := set.ErrorToCodeFunc(rpcInfo.Err)
	event := rkgrpcctx.GetEvent(ctx)
	event.AddErr(rpcInfo.Err)
	endTime := time.Now()
	elapsed := endTime.Sub(event.GetStartTime())

	fields := make([]zap.Field, 0)

	// Check whether context is cancelled from server
	select {
	case <-ctx.Done():
		event.AddErr(ctx.Err())
		fields = append(fields, zap.NamedError("serverError", ctx.Err()))
	default:
		break
	}

	// Extract request id and log it
	incomingRequestIds := rkgrpcctx.GetRequestIdsFromIncomingMD(ctx)
	fields = append(fields,
		zap.String("resCode", code.String()),
		zap.Time("endTime", time.Now()),
		zap.Int64("elapsedNano", elapsed.Nanoseconds()),
		zap.Strings("incomingRequestId", incomingRequestIds))

	rkgrpcctx.SetZapLogger(ctx, rkgrpcctx.GetZapLogger(ctx).With(
		zap.Strings("incomingRequestId", incomingRequestIds)))

	event.AddFields(fields...)
	if len(event.GetEventId()) < 1 {
		event.SetEventId("fakeId")
	}
	event.SetEndTime(endTime)

	event.WriteLog()
}
