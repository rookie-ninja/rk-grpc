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
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/tracing/telemetry"
	"google.golang.org/grpc"
	"log"
	"net"
)

// In this example, we will create a simple grpc server and enable trace interceptor.
// Then, we will try to send requests to server.
func main() {
	// ****************************************
	// ********** Create Exporter *************
	// ****************************************

	// Export trace to stdout
	exporter := rkgrpctrace.CreateFileExporter("stdout")

	// Export trace to local file system
	// exporter := rkgrpctrace.CreateFileExporter("logs/trace.log")

	// Export trace to jaeger collector
	// exporter := rkgrpctrace.CreateJaegerExporter("localhost:14268", "", "")

	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			rkgrpclog.UnaryServerInterceptor(),
			rkgrpctrace.UnaryServerInterceptor(
			// Entry name and entry type will be used for distinguishing interceptors. Recommended.
			// rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
			//
			// Provide an exporter.
			rkgrpctrace.WithExporter(exporter),
			//
			// Provide propagation.TextMapPropagator
			// rkgrpctrace.WithPropagator(<propagator>),
			//
			// Provide SpanProcessor
			// rkgrpctrace.WithSpanProcessor(<span processor>),
			//
			// Provide TracerProvider
			// rkgrpctrace.WithTracerProvider(<trace provider>),
			),
		),
	}

	// 1: Create grpc server
	server := startGreeterServer(opts...)
	defer server.GracefulStop()

	// 2: Wait for ctrl-C to shutdown server
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
