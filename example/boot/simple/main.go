// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/boot"
	proto "github.com/rookie-ninja/rk-grpc/example/boot/simple/api/gen/v1"
	"google.golang.org/grpc"
)

func main() {
	// Bootstrap basic entries from boot config.
	rkentry.RegisterInternalEntriesFromConfig("example/boot/simple/boot.yaml")

	// Bootstrap grpc entry from boot config
	res := rkgrpc.RegisterGrpcEntriesWithConfig("example/boot/simple/boot.yaml")

	// Get GrpcEntry
	grpcEntry := res["greeter"].(*rkgrpc.GrpcEntry)
	// Register gRPC server
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		proto.RegisterGreeterServer(server, &GreeterServer{})
	})
	// Register grpc-gateway func
	grpcEntry.AddRegFuncGw(proto.RegisterGreeterHandlerFromEndpoint)

	// Bootstrap grpc entry
	grpcEntry.Bootstrap(context.Background())

	// Wait for shutdown signal
	rkentry.GlobalAppCtx.WaitForShutdownSig()

	// Interrupt gin entry
	grpcEntry.Interrupt(context.Background())
}

// GreeterServer Implementation of GreeterServer.
type GreeterServer struct{}

// SayHello Handle SayHello method.
func (server *GreeterServer) Greeter(context.Context, *proto.GreeterRequest) (*proto.GreeterResponse, error) {
	return &proto.GreeterResponse{}, nil
}
