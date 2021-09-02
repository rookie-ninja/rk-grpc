// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpcauth

import (
	"fmt"
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Create new unary client interceptor.
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		// 1: Before invoking
		clientBefore(ctx, set)

		opts = append(opts, grpc.Header(rkgrpcinter.GetIncomingHeadersOfClient(ctx)))

		// Add outgoing md to context
		ctx = rkgrpcinter.MergeToOutgoingMD(ctx, *rkgrpcinter.GetOutgoingHeadersOfClient(ctx))

		// 2: Invoking
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

// Create new stream client interceptor.
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	set := newOptionSet(rkgrpcinter.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		rkgrpcinter.AddToClientContextPayload(ctx, rkgrpcinter.RpcEntryNameKey, set.EntryName)

		// Before invoking
		clientBefore(ctx, set)

		// Add outgoing md to context
		ctx = rkgrpcinter.MergeToOutgoingMD(ctx, *rkgrpcinter.GetOutgoingHeadersOfClient(ctx))

		// Invoking
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// Handle logic before handle requests.
func clientBefore(ctx context.Context, set *optionSet) {
	for k := range set.BasicAccounts {
		val := fmt.Sprintf("%s %s", typeBasic, k)
		rkgrpcctx.AddHeaderToServer(ctx, rkgrpcinter.RpcAuthorizationHeaderKey, val)
	}

	for k := range set.ApiKey {
		rkgrpcctx.AddHeaderToServer(ctx, rkgrpcinter.RpcApiKeyHeaderKey, k)
	}
}
