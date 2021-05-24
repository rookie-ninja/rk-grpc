// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"path"
	"strings"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcctx.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		// Before invoking
		incomingMD := rkgrpcctx.GetIncomingMD(ctx)
		outgoingMD := rkgrpcctx.GetOutgoingMD(ctx)

		// Parse remote address
		remoteIp, remotePort, _ := rkgrpcctx.GetRemoteAddressSet(ctx)

		grpcService, grpcMethod, gwMethod, gwPath := parseRpcPath(ctx, info.FullMethod)

		ctx = rkgrpcctx.ContextWithPayload(ctx,
			rkgrpcctx.WithEntryName(set.EntryName),
			rkgrpcctx.WithIncomingMD(incomingMD),
			rkgrpcctx.WithOutgoingMD(outgoingMD),
			rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
				GrpcService: grpcService,
				GrpcMethod:  grpcMethod,
				GwMethod:    gwMethod,
				GwPath:      gwPath,
				Type:        rkgrpcctx.RpcTypeUnaryServer,
				RemoteIp:    remoteIp,
				RemotePort:  remotePort,
			}))

		// Invoking
		resp, err := handler(ctx, req)

		// After invoking
		if rpcInfo := rkgrpcctx.GetRpcInfo(ctx); rpcInfo != nil {
			rpcInfo.Err = err
		}

		grpc.SetHeader(ctx, rkgrpcctx.GetOutgoingMD(ctx))

		return resp, err
	}
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcctx.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		ctx := wrappedStream.WrappedContext

		incomingMD := rkgrpcctx.GetIncomingMD(ctx)
		outgoingMD := rkgrpcctx.GetOutgoingMD(ctx)

		remoteIp, remotePort, _ := rkgrpcctx.GetRemoteAddressSet(ctx)

		// Parse rpc path info including gateway
		grpcService, grpcMethod, gwMethod, gwPath := parseRpcPath(ctx, info.FullMethod)

		ctx = rkgrpcctx.ContextWithPayload(ctx,
			rkgrpcctx.WithEntryName(set.EntryName),
			rkgrpcctx.WithIncomingMD(incomingMD),
			rkgrpcctx.WithOutgoingMD(outgoingMD),
			rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
				GrpcService: grpcService,
				GrpcMethod:  grpcMethod,
				GwMethod:    gwMethod,
				GwPath:      gwPath,
				Type:        rkgrpcctx.RpcTypeStreamServer,
				RemoteIp:    remoteIp,
				RemotePort:  remotePort,
			}))

		// Invoking
		err := handler(srv, wrappedStream)

		// After invoking
		if rpcInfo := rkgrpcctx.GetRpcInfo(ctx); rpcInfo != nil {
			rpcInfo.Err = err
		}

		grpc.SetHeader(ctx, rkgrpcctx.GetOutgoingMD(ctx))

		return err
	}
}

func parseRpcPath(ctx context.Context, fullMethod string) (grpcService, grpcMethod, gwMethod, gwPath string) {
	// Parse rpc path info including gateway
	grpcService, grpcMethod, gwMethod, gwPath = "unknown", "unknown", "unknown", "unknown"

	if v := strings.TrimPrefix(path.Dir(fullMethod), "/"); len(v) > 0 {
		grpcService = v
	}

	if v := path.Base(fullMethod); len(v) > 0 {
		grpcMethod = v
	}

	if tokens := rkgrpcctx.GetValueFromIncomingMD(ctx, "x-forwarded-method"); len(tokens) > 0 {
		gwMethod = tokens[0]
	}

	if tokens := rkgrpcctx.GetValueFromIncomingMD(ctx, "x-forwarded-path"); len(tokens) > 0 {
		gwPath = tokens[0]
	}

	return grpcService, grpcMethod, gwMethod, gwPath
}
