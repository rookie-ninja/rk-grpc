// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"path"
	"strings"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Before invoking
		incomingMD := rkgrpcctx.GetIncomingMD(ctx)
		outgoingMD := rkgrpcctx.GetOutgoingMD(ctx)

		remoteIp, remotePort, _ := getRemoteAddressSet(ctx)
		grpcService, grpcMethod := getGrpcInfo(info.FullMethod)
		gwMethod, gwPath, gwScheme, gwUserAgent := getGwInfo(incomingMD)

		ctx = rkgrpcctx.ToRkContext(ctx,
			rkgrpcctx.WithEntryName(set.EntryName),
			rkgrpcctx.WithIncomingMD(incomingMD),
			rkgrpcctx.WithOutgoingMD(outgoingMD),
			rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
				GrpcService: grpcService,
				GrpcMethod:  grpcMethod,
				GwMethod:    gwMethod,
				GwPath:      gwPath,
				GwScheme:    gwScheme,
				GwUserAgent: gwUserAgent,
				Type:        RpcTypeUnaryServer,
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
	set := newOptionSet(RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		ctx := wrappedStream.WrappedContext

		incomingMD := rkgrpcctx.GetIncomingMD(ctx)
		outgoingMD := rkgrpcctx.GetOutgoingMD(ctx)

		remoteIp, remotePort, _ := getRemoteAddressSet(ctx)
		grpcService, grpcMethod := getGrpcInfo(info.FullMethod)
		gwMethod, gwPath, gwScheme, gwUserAgent := getGwInfo(incomingMD)

		ctx = rkgrpcctx.ToRkContext(ctx,
			rkgrpcctx.WithEntryName(set.EntryName),
			rkgrpcctx.WithIncomingMD(incomingMD),
			rkgrpcctx.WithOutgoingMD(outgoingMD),
			rkgrpcctx.WithRpcInfo(&rkgrpcctx.RpcInfo{
				GrpcService: grpcService,
				GrpcMethod:  grpcMethod,
				GwMethod:    gwMethod,
				GwPath:      gwPath,
				GwScheme:    gwScheme,
				GwUserAgent: gwUserAgent,
				Type:        RpcTypeStreamServer,
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

func getGwInfo(md metadata.MD) (gwMethod, gwPath, gwScheme, gwUserAgent string) {
	gwMethod, gwPath, gwScheme, gwUserAgent = "unknown", "unknown", "unknown", "unknown"

	if tokens := md["x-forwarded-method"]; len(tokens) > 0 {
		gwMethod = tokens[0]
	}

	if tokens := md["x-forwarded-path"]; len(tokens) > 0 {
		gwPath = tokens[0]
	}

	if tokens := md["x-forwarded-scheme"]; len(tokens) > 0 {
		gwScheme = tokens[0]
	}

	if tokens := md["x-forwarded-user-agent"]; len(tokens) > 0 {
		gwUserAgent = tokens[0]
	}

	return gwMethod, gwPath, gwScheme, gwUserAgent
}

func getGrpcInfo(fullMethod string) (grpcService, grpcMethod string) {
	// Parse rpc path info including gateway
	grpcService, grpcMethod = "unknown", "unknown"

	if v := strings.TrimPrefix(path.Dir(fullMethod), "/"); len(v) > 0 {
		grpcService = v
	}

	if v := strings.TrimPrefix(path.Base(fullMethod), "/"); len(v) > 0 {
		grpcMethod = v
	}

	return grpcService, grpcMethod
}
