// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-grpc/example/interceptor/proto/gen"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"log"
)

// In this example, we will create a simple grpc client and enable log interceptor.
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
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.DialOption{
		grpc.WithChainStreamInterceptor(
			rkgrpclog.StreamClientInterceptor(
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
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// 1: Create grpc client
	conn, client := createChatClient(opts...)
	defer conn.Close()

	// 2: Recommended: Wrap context, this is recommended.
	ctx := rkgrpcctx.WrapContext(context.Background())

	// 3: Call server
	stream, err := client.Say(ctx)
	if err != nil {
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to chat with server.", zap.Error(err))
	}

	// 4: Send "Hi" and "Nice to meet you!" to server
	if err := stream.Send(&proto.ServerMessage{Message: "Hi!"}); err != nil {
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to send to server.", zap.Error(err))
	}
	if err := stream.Send(&proto.ServerMessage{Message: "Nice to meet you!"}); err != nil {
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to send to server.", zap.Error(err))
	}
	// 5: Close stream
	stream.CloseSend()

	// 6: Receiving server response
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			rkgrpcctx.GetLogger(ctx).Fatal("Failed to receive message from client.", zap.Error(err))
		}

		rkgrpcctx.GetLogger(ctx).Info(fmt.Sprintf("[From server]: %s", in.Message))
	}

	// Streaming client is a little bit tricky.
	// It is not an easy work to get headers sent from server while receiving message
	// since client stream interceptor will finish before client start receiving message.
	//
	// As a result, what event will log is the time before Recv() start to be called.
	// No request id nor trace id would be logged since we are unable to call stream.Header() function which would be
	// blocked until stream.Recv() has been called.
	//
	// We believe it is not a good idea to wrap client stream or do anything tricky with stream.
	//
	// If user hope to log request id and trace id into event, user need to call bellow function as soon as stream.Header()
	// is ready.
	// The downside is you will get multiple event logs with same event id.
	//
	// rkgrpcctx.FinishClientStream(ctx, stream)
}

// Create grpc client.
func createChatClient(opts ...grpc.DialOption) (*grpc.ClientConn, proto.ChatClient) {
	// 1: Set up a connection to the server.
	conn, err := grpc.DialContext(context.Background(), "localhost:8080", opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// 2: Create grpc client
	client := proto.NewChatClient(conn)

	return conn, client
}
