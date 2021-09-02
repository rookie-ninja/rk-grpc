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
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
)

// In this example, we will create a simple grpc client and enable panic interceptor.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			rkgrpclog.UnaryClientInterceptor(),
			// Add panic interceptor
			rkgrpcpanic.UnaryClientInterceptor(),
		),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// 1: Create grpc client
	conn, client := createGreeterClient(opts...)
	defer conn.Close()

	// 2: Wrap context, this is required in order to use bellow features easily.
	ctx := rkgrpcctx.WrapContext(context.Background())

	// 3: Call server
	if resp, err := client.SayHello(ctx, &proto.HelloRequest{Name: "rk-dev"}); err != nil {
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to send request to server.", zap.Error(err))
	} else {
		rkgrpcctx.GetLogger(ctx).Info(fmt.Sprintf("[Message]: %s", resp.Message))
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
