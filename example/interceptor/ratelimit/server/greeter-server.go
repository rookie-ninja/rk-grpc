// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	rkmidlimit "github.com/rookie-ninja/rk-entry/middleware/ratelimit"
	proto "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/testdata"
	rkgrpclog "github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/ratelimit"
	"google.golang.org/grpc"
	"log"
	"net"
)

// In this example, we will create a simple grpc server and enable rate limit interceptor.
// Then, we will try to send requests to server.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			rkgrpclog.UnaryServerInterceptor(),
			rkgrpclimit.UnaryServerInterceptor(
				// Entry name and entry type will be used for distinguishing interceptors. Recommended.
				// rkmidlimit.WithEntryNameAndType("greeter", "grpc"),
				//
				// Provide algorithm, rkmidlimit.LeakyBucket and rkmidlimit.TokenBucket was available, default is TokenBucket.
				rkmidlimit.WithAlgorithm(rkmidlimit.LeakyBucket),
				//
				// Provide request per second, if provide value of zero, then no requests will be pass through and user will receive an error with
				// resource exhausted.
				rkmidlimit.WithReqPerSec(0),
				//
				// Provide request per second with path name.
				// The name should be gRPC full method name. if provide value of zero,
				// then no requests will be pass through and user will receive an error with resource exhausted.
				// rkmidlimit.WithReqPerSecByPath("/Greeter/SayHello", 0),
				//
				// Provide user function of limiter
				//rkmidlimit.WithGlobalLimiter(func() error {
				//	 return nil
				//}),
				//
				// Provide user function of limiter by path name.
				// The name should be gRPC full method name.
				// rkmidlimit.WithLimiterByPath("/Greeter/SayHello", func() error {
				//	 return nil
				// }),
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
