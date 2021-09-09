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
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"google.golang.org/grpc"
	"log"
	"net"
)

// In this example, we will create a simple gRpc server and enable log interceptor.
// Then, we will try to send requests to server and monitor what kinds of logging we would get.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			rkgrpclog.UnaryServerInterceptor(
			// Entry name and entry type will be used for distinguishing interceptors. Recommended.
			// rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
			//
			// Zap logger would be logged as JSON format.
			// rkgrpclog.WithZapLoggerEncoding(rkgrpclog.ENCODING_JSON),
			//
			// Event logger would be logged as JSON format.
			// rkgrpclog.WithEventLoggerEncoding(rkgrpclog.ENCODING_JSON),
			//
			// Zap logger would be logged to specified path.
			// rkgrpclog.WithZapLoggerOutputPaths("logs/server-zap.log"),
			//
			// Event logger would be logged to specified path.
			// rkgrpclog.WithEventLoggerOutputPaths("logs/server-event.log"),
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
	// ******************************************
	// ********** rpc-scoped logger *************
	// ******************************************
	//
	// RequestId will be printed if enabled by bellow codes.
	// 1: Enable rkgrpcextension.UnaryServerInterceptor() in server side.
	// 2: rkgrpcctx.AddHeaderToClient(ctx, rkgrpcctx.RequestIdKey, rkcommon.GenerateRequestId())
	//
	rkgrpcctx.GetLogger(ctx).Info("Received request from client.")

	// *******************************************
	// ********** rpc-scoped event  *************
	// *******************************************
	//
	// Get rkquery.Event which would be printed as soon as request finish.
	// User can call any Add/Set/Get functions on rkquery.Event
	//
	// rkgrpcctx.GetEvent(ctx).AddPair("rk-key", "rk-value")

	// *********************************************
	// ********** Get incoming headers *************
	// *********************************************
	//
	// Read headers sent from client or grpc-gateway.
	//
	// for k, v := range rkgrpcctx.GetIncomingHeaders(ctx) {
	//	 fmt.Println(fmt.Sprintf("%s: %s", k, v))
	// }

	// *********************************************************
	// ********** Add headers will send to client **************
	// *********************************************************
	//
	// Send headers to client with this function
	//
	// rkgrpcctx.AddHeaderToClient(ctx, "from-server", "value")

	// ***********************************************
	// ********** Get and log request id *************
	// ***********************************************
	//
	// RequestId will be printed on both client and server side.
	//
	// rkgrpcctx.AddHeaderToClient(ctx, rkgrpcctx.RequestIdKey, rkcommon.GenerateRequestId())

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
