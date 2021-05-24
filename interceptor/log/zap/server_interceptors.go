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
	"path"
	"time"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcctx.RpcTypeUnaryServer, opts...)

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
	set := newOptionSet(rkgrpcctx.RpcTypeStreamServer, opts...)

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

func serverBefore(ctx context.Context, options *optionSet) context.Context {
	event := options.EventLoggerEntry.GetEventFactory().CreateEvent()
	event.SetStartTime(time.Now())

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)

	// Add request ids from remote side
	incomingRequestIds := rkgrpcctx.GetRequestIdsFromIncomingMD(ctx)

	fields := []zap.Field{
		rkgrpcctx.Realm,
		rkgrpcctx.Region,
		rkgrpcctx.AZ,
		rkgrpcctx.Domain,
		zap.String("appVersion", rkentry.GlobalAppCtx.GetAppInfoEntry().Version),
		zap.String("appName", rkentry.GlobalAppCtx.GetAppInfoEntry().AppName),
		rkgrpcctx.LocalIp,
		zap.String("grpcService", rpcInfo.GrpcService),
		zap.String("grpcMethod", rpcInfo.GrpcMethod),
		zap.String("grpcType", rpcInfo.Type),
		zap.String("gwMethod", rpcInfo.GwMethod),
		zap.String("gwPath", rpcInfo.GwPath),
		zap.Strings("incomingRequestId", incomingRequestIds),
		zap.Time("startTime", event.GetStartTime()),
	}

	logger := options.ZapLoggerEntry.GetLogger().With(
		zap.Strings("incomingRequest_Id", incomingRequestIds))

	remoteAddressSet := rkgrpcctx.GetRemoteAddressSetAsFields(ctx)
	fields = append(fields, remoteAddressSet...)
	event.SetRemoteAddr(remoteAddressSet[0].String)
	event.SetOperation(path.Base(rpcInfo.GrpcMethod))

	if d, ok := ctx.Deadline(); ok {
		event.AddErr(ctx.Err())
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	event.AddFields(fields...)

	return rkgrpcctx.ContextWithPayload(ctx,
		rkgrpcctx.WithEvent(event),
		rkgrpcctx.WithZapLogger(logger),
	)
}

func serverAfter(ctx context.Context, options *optionSet) {
	event := rkgrpcctx.GetEvent(ctx)

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)
	event.AddErr(rpcInfo.Err)
	code := options.ErrorToCodeFunc(rpcInfo.Err)
	endTime := time.Now()
	elapsed := endTime.Sub(event.GetStartTime())

	fields := make([]zap.Field, 0)

	// Check whether context is cancelled from client
	select {
	case <-ctx.Done():
		event.AddErr(ctx.Err())
		fields = append(fields, zap.NamedError("clientError", ctx.Err()))
	default:
		break
	}

	// extract request id and log it
	fields = append(fields,
		zap.String("resCode", code.String()),
		zap.Time("endTime", endTime),
		zap.Int64("elapsedNano", elapsed.Nanoseconds()),
		zap.Strings("outgoingRequestId", rkgrpcctx.GetRequestIdsFromOutgoingMD(ctx)),
	)

	event.AddFields(fields...)
	if len(event.GetEventId()) < 1 {
		event.SetEventId("fakeId")
	}
	event.SetEndTime(endTime)

	event.WriteLog()
}
