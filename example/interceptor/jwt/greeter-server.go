// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	proto "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/testdata"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/jwt"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

// In this example, we will create a simple grpc server and enable auth interceptor.
// Then, we will try to send requests to server.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			rkgrpclog.UnaryServerInterceptor(),
			rkgrpcjwt.UnaryServerInterceptor(
				rkgrpcjwt.WithSigningKey([]byte("my-secret")),
			),
		),
	}

	// 1: Create grpc server
	server := startGreeterServer(opts...)
	defer server.GracefulStop()

	// 2: Wait for ctrl-C to shutdown server
	rkentry.GlobalAppCtx.WaitForShutdownSig()
}

// GreeterServer Implementation of GreeterServer.
type GreeterServer struct{}

// SayHello Handle SayHello method.
func (server *GreeterServer) SayHello(ctx context.Context, request *proto.HelloRequest) (*proto.HelloResponse, error) {
	return &proto.HelloResponse{
		Message: fmt.Sprintf("Is token valid:%v!", rkgrpcctx.GetJwtToken(ctx).Valid),
	}, nil
}

// Create and start server.
func startGreeterServer(opts ...grpc.ServerOption) *grpc.Server {
	// 1: Create listener with port 8080
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 2: Create grpc server with grpc.ServerOption
	server := grpc.NewServer(opts...)

	reflection.Register(server)

	// 3: Register server to proto
	proto.RegisterGreeterServer(server, &GreeterServer{})

	// 4: Start server
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return server
}
