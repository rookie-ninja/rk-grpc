// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	api "github.com/rookie-ninja/rk-grpc/boot/api/third_party/gen/v1"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
)

// In this example, we will create a simple gRpc client and enable RK style logging interceptor.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			rkgrpclog.UnaryClientInterceptor(),
		),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// 1: Create grpc client
	conn, client := createCommonServiceClient(opts...)
	defer conn.Close()

	// 2: Wrap context, this is required in order to use bellow features easily.
	ctx := rkgrpcctx.WrapContext(context.Background())
	// Add header to make proxy request to test server
	ctx = metadata.AppendToOutgoingContext(ctx, "domain", "test")

	// 3: Call server
	if resp, err := client.Healthy(ctx, &api.HealthyRequest{}); err != nil {
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to send request to server.", zap.Error(err))
	} else {
		rkgrpcctx.GetLogger(ctx).Info(fmt.Sprintf("[Message]: %s", resp.String()))
	}
}

func createCommonServiceClient(opts ...grpc.DialOption) (*grpc.ClientConn, api.RkCommonServiceClient) {
	// 1: Set up a connection to the server.
	conn, err := grpc.DialContext(context.Background(), "localhost:8080", opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// 2: Create grpc client
	client := api.NewRkCommonServiceClient(conn)

	return conn, client
}
