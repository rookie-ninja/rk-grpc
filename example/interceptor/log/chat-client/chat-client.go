// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	api "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/testdata"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"log"
)

var logger, _ = zap.NewDevelopment()

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
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// 1: Create grpc client
	conn, client := createChatClient(opts...)
	defer conn.Close()

	// 2: Call server
	stream, err := client.Say(context.TODO())
	if err != nil {
		logger.Fatal("Failed to chat with server.", zap.Error(err))
	}

	// 3: Send "Hi" and "Nice to meet you!" to server
	if err := stream.Send(&api.ServerMessage{Message: "Hi!"}); err != nil {
		logger.Fatal("Failed to send to server.", zap.Error(err))
	}
	if err := stream.Send(&api.ServerMessage{Message: "Nice to meet you!"}); err != nil {
		logger.Fatal("Failed to send to server.", zap.Error(err))
	}
	// 4: Close stream
	stream.CloseSend()

	// 5: Receiving server response
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			logger.Fatal("Failed to receive message from client.", zap.Error(err))
		}

		logger.Info(fmt.Sprintf("[From server]: %s", in.Message))
	}
}

// Create grpc client.
func createChatClient(opts ...grpc.DialOption) (*grpc.ClientConn, api.ChatClient) {
	// 1: Set up a connection to the server.
	conn, err := grpc.DialContext(context.Background(), "localhost:8080", opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// 2: Create grpc client
	client := api.NewChatClient(conn)

	return conn, client
}
