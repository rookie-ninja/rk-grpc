// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc_panic

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(in ...PanicHandler) grpc.UnaryServerInterceptor {
	handlers = append(handlers, in...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer handleCrash(ctx, func(ctx context.Context, r interface{}) {
			err = toError(r)
		})

		return handler(ctx, req)
	}
}

func StreamServerInterceptor(in ...PanicHandler) grpc.StreamServerInterceptor {
	handlers = append(handlers, in...)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer handleCrash(stream.Context(), func(ctx context.Context, r interface{}) {
			err = toError(r)
		})

		return handler(srv, stream)
	}
}

func toError(r interface{}) error {
	return status.Errorf(codes.Internal, "Panic by %v", r)
}

func handleCrash(ctx context.Context, handler PanicHandler) {
	if r := recover(); r != nil {
		handler(ctx, r)

		for _, fn := range handlers {
			println(len(handlers))
			fn(ctx, r)
		}
	}
}

type panic struct{}

func (e panic) Error() string {
	return "panic captured"
}
