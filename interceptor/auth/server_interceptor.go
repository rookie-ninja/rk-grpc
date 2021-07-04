// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcauth

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// Create new unary server interceptor.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryServer, opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx = rkgrpcinter.WrapContextForServer(ctx)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(ctx, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// Before invoking
		if err := serverBefore(ctx, set, info.FullMethod); err != nil {
			return nil, err
		}

		// Invoking
		return handler(ctx, req)
	}
}

// Create new stream server interceptor.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamServer, opts...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Before invoking
		wrappedStream := rkgrpcctx.WrapServerStream(stream)
		wrappedStream.WrappedContext = rkgrpcinter.WrapContextForServer(wrappedStream.WrappedContext)

		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcEntryNameKey, set.EntryName)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcTypeKey, rkgrpcinter.RpcTypeUnaryServer)
		rkgrpcinter.AddToServerContextPayload(wrappedStream.WrappedContext, rkgrpcinter.RpcMethodKey, info.FullMethod)

		// Before invoking
		if err := serverBefore(wrappedStream.WrappedContext, set, info.FullMethod); err != nil {
			return err
		}

		// Invoking
		return handler(srv, wrappedStream)
	}
}

// Handle logic before handle requests.
func serverBefore(ctx context.Context, set *optionSet, method string) error {
	if !set.ShouldAuth(method) {
		return nil
	}

	headers := rkgrpcctx.GetIncomingHeaders(ctx)

	authorizationHeader := headers.Get(rkgrpcinter.RpcAuthorizationHeaderKey)
	apiKeyHeader := headers.Get(rkgrpcinter.RpcApiKeyHeaderKey)

	if len(authorizationHeader) > 0 {
		// Basic auth type
		tokens := strings.SplitN(authorizationHeader[0], " ", 2)
		if len(tokens) != 2 {
			return status.Error(codes.Unauthenticated, `Invalid Basic Auth format`)
		}

		if !set.Authorized(tokens[0], tokens[1]) {
			if tokens[0] == typeBasic {
				rkgrpcctx.AddHeaderToClient(ctx, "WWW-Authenticate", fmt.Sprintf(`%s realm="%s"`, typeBasic, set.BasicRealm))
			}

			return status.Error(codes.Unauthenticated, `Invalid credential`)
		} else {
			return nil
		}
	} else if len(apiKeyHeader) > 0 {
		// API key auth type
		if !set.Authorized(typeApiKey, apiKeyHeader[0]) {
			return status.Error(codes.Unauthenticated, `Invalid X-API-Key`)
		} else {
			return nil
		}
	} else {
		authHeaders := []string{}
		if len(set.BasicAccounts) > 0 {
			rkgrpcctx.AddHeaderToClient(ctx, "WWW-Authenticate", fmt.Sprintf(`%s realm="%s"`, typeBasic, set.BasicRealm))
			authHeaders = append(authHeaders, "Basic Auth")
		}
		if len(set.ApiKey) > 0 {
			authHeaders = append(authHeaders, "X-API-Key")
		}

		errMsg := fmt.Sprintf("Missing authorization, provide one of bellow auth header:[%s]", strings.Join(authHeaders, ","))

		return status.Error(codes.Unauthenticated, errMsg)
	}
}
