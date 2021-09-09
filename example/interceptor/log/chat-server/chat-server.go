// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	api "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/gen"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
)

// In this example, we will create a simple grpc server and enable log interceptor.
// Then, we will try to send requests to server and monitor what kinds of logging we would get.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.ServerOption{
		grpc.ChainStreamInterceptor(
			rkgrpclog.StreamServerInterceptor(
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
	server := startChatServer(opts...)
	defer server.GracefulStop()

	// 2: Wait for ctrl-C to shutdown server
	rkentry.GlobalAppCtx.WaitForShutdownSig()
}

// ChatServer Implementation of ChatServer.
//
// The bidirectional communication between client and server.
//
//     +--------+					+--------+
//     | Client |					| Server |
//     +--------+					+--------+
// 	       |						     |
//         |             Hi!             |
//         |-------------------------->>>|
// 	       |						     |
//         |      Nice to meet you!      |
//         |-------------------------->>>|
// 	       |						     |
//         |             Hi!             |
//         |<<<--------------------------|
// 	       |						     |
//         |    Nice to meet you too!    |
//         |<<<--------------------------|
type ChatServer struct{}

// Say Implementation of Say().
func (server *ChatServer) Say(stream api.Chat_SayServer) error {
	for {
		in, err := stream.Recv()

		if err == io.EOF {
			if err := stream.Send(&api.ClientMessage{Message: "Hi!"}); err != nil {
				return err
			}
			if err := stream.Send(&api.ClientMessage{Message: "Nice to meet you too!"}); err != nil {
				return err
			}

			return nil
		}

		if err != nil {
			return err
		}

		rkgrpcctx.GetLogger(stream.Context()).Info(fmt.Sprintf("[From client]: %s", in.Message))
	}
}

// Create and start server.
func startChatServer(opts ...grpc.ServerOption) *grpc.Server {
	// 1: Create listener with port 8080
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 2: Create grpc server with grpc.ServerOption
	server := grpc.NewServer(opts...)

	// 3: Register server to proto
	api.RegisterChatServer(server, &ChatServer{})

	// 4: Start server
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return server
}
