// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpclog

import (
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"net"
	"time"
)

// Create new unary client interceptor.
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		// 1: Before invoking
		ctx = clientBefore(ctx, set, method, rkgrpcinter.RpcTypeUnaryClient, cc.Target())

		opts = append(opts, grpc.Header(rkgrpcinter.GetIncomingHeadersOfClient(ctx)))

		// Add outgoing md to context
		ctx = rkgrpcinter.MergeToOutgoingMD(ctx, *rkgrpcinter.GetOutgoingHeadersOfClient(ctx))

		// 2: Invoking
		err := invoker(ctx, method, req, resp, cc, opts...)

		// 3: After invoking
		clientAfter(ctx, set, err)

		return err
	}
}

// Create new stream client interceptor.
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		// Before invoking
		ctx = clientBefore(ctx, set, method, rkgrpcinter.RpcTypeStreamClient, cc.Target())

		// Add outgoing md to context
		ctx = rkgrpcinter.MergeToOutgoingMD(ctx, *rkgrpcinter.GetOutgoingHeadersOfClient(ctx))

		// Invoking
		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		// After invoking
		clientAfter(ctx, set, err)

		return clientStream, err
	}
}

// Handle logic before handle requests.
func clientBefore(ctx context.Context, set *optionSet, method, grpcType, remoteEndpoint string) context.Context {
	event := set.eventLoggerEntry.GetEventFactory().CreateEvent(
		rkquery.WithZapLogger(set.eventLoggerOverride),
		rkquery.WithEncoding(set.eventLoggerEncoding),
		rkquery.WithAppName(rkentry.GlobalAppCtx.GetAppInfoEntry().AppName),
		rkquery.WithAppVersion(rkentry.GlobalAppCtx.GetAppInfoEntry().Version),
		rkquery.WithEntryName(set.EntryName),
		rkquery.WithEntryType(set.EntryType))
	event.SetStartTime(time.Now())

	remoteIp, remotePort, _ := net.SplitHostPort(remoteEndpoint)
	grpcService, grpcMethod := rkgrpcinter.GetGrpcInfo(method)

	event.SetRemoteAddr(remoteIp + ":" + remotePort)
	event.SetOperation(method)

	payloads := []zap.Field{
		zap.String("remoteIp", remoteIp),
		zap.String("remotePort", remotePort),
		zap.String("grpcService", grpcService),
		zap.String("grpcMethod", grpcMethod),
		zap.String("grpcType", grpcType),
	}

	if d, ok := ctx.Deadline(); ok {
		payloads = append(payloads, zap.String("deadline", d.Format(time.RFC3339)))
	}

	event.AddPayloads(payloads...)

	// insert logger and event
	rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEventKey, event)
	rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcLoggerKey, set.ZapLogger)

	return ctx
}

// Handle logic after handle requests.
func clientAfter(ctx context.Context, set *optionSet, err error) {
	event := rkgrpcctx.GetEvent(ctx)
	event.AddErr(err)
	code := status.Code(err)
	endTime := time.Now()

	// Check whether context is cancelled from server
	select {
	case <-ctx.Done():
		event.AddErr(ctx.Err())
	default:
		break
	}

	// Read X-Request-Id header sent from server if exists
	incomingMD := rkgrpcinter.GetIncomingHeadersOfClient(ctx)
	if v := incomingMD.Get(rkgrpcctx.RequestIdKey); len(v) > 0 {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcctx.RequestIdKey, v[len(v)-1])
	}

	// Extract request id and log it
	incomingRequestId := rkgrpcctx.GetRequestId(ctx)

	if len(incomingRequestId) > 0 {
		event.SetEventId(incomingRequestId)
		event.SetRequestId(incomingRequestId)
	}

	traceId := rkgrpcctx.GetTraceId(ctx)
	if len(traceId) > 0 {
		event.SetTraceId(traceId)
	}

	event.SetResCode(code.String())
	event.SetEndTime(endTime)
	event.Finish()
}
