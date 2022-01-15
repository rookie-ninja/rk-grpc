// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	proto "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/testdata"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
)

var logger, _ = zap.NewDevelopment()

// In this example, we will create a simple grpc client and enable metrics interceptor.
func main() {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// 1: Create grpc client
	conn, client := createGreeterClient(opts...)
	defer conn.Close()

	// 2: Call server
	if resp, err := client.SayHello(context.TODO(), &proto.HelloRequest{Name: "rk-dev"}); err != nil {
		logger.Fatal("Failed to send request to server.", zap.Error(err))
	} else {
		logger.Info(fmt.Sprintf("[Message]: %s", resp.Message))
	}
}

func createGreeterClient(opts ...grpc.DialOption) (*grpc.ClientConn, proto.GreeterClient) {
	// 1: Set up a connection to the server.
	conn, err := grpc.DialContext(context.Background(), "localhost:8080", opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// 2: Create grpc client
	client := proto.NewGreeterClient(conn)

	return conn, client
}
