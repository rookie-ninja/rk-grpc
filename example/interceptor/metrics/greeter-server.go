// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/example/interceptor/proto/gen"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-prom"
	"google.golang.org/grpc"
	"log"
	"net"
)

// In this example, we will create a simple grpc server and enable metrics interceptor.
// Then, we will try to send requests to server and monitor what kinds of logging we would get.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			// Add metrics interceptor
			rkgrpcmetrics.UnaryServerInterceptor(
			// Entry name and entry type will be used for distinguishing interceptors. Recommended.
			// rkgrpcmetrics.WithEntryNameAndType("greeter", "grpc"),
			//
			// Provide new prometheus registerer.
			// Default value is prometheus.DefaultRegisterer
			// rkgrpcmetrics.WithRegisterer(prometheus.NewRegistry()),
			),
		),
	}

	// 2: Start prometheus client
	// By default, we will start prometheus client at localhost:1608/metrics
	promEntry := rkprom.RegisterPromEntry(rkprom.WithPort(8081))
	promEntry.Bootstrap(context.Background())
	defer promEntry.Interrupt(context.Background())

	// 3: Create grpc server
	server := startGreeterServer(opts...)
	defer server.GracefulStop()

	// 3: Wait for ctrl-C to shutdown server
	rkentry.GlobalAppCtx.WaitForShutdownSig()
}

// Implementation of GreeterServer.
type GreeterServer struct{}

// Handle SayHello method.
func (server *GreeterServer) SayHello(ctx context.Context, request *proto.HelloRequest) (*proto.HelloResponse, error) {
	return &proto.HelloResponse{
		Message: fmt.Sprintf("Hello %s!", request.GetName()),
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
