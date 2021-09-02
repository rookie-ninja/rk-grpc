// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/example/interceptor/proto/gen"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-prom"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
)

// In this example, we will create a simple grpc client and enable metrics interceptor.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			// Add metrics interceptor
			rkgrpcmetrics.UnaryClientInterceptor(
			// Entry name and entry type will be used for distinguishing interceptors. Recommended.
			// rkgrpcmetrics.WithEntryNameAndType("greeter", "grpc"),
			//
			// Provide new prometheus registerer.
			// Default value is prometheus.DefaultRegisterer
			// rkgrpcmetrics.WithRegisterer(prometheus.NewRegistry()),
			),
		),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// 1: Start prometheus client
	// By default, we will start prometheus client at localhost:1608/metrics
	promEntry := rkprom.RegisterPromEntry(rkprom.WithPort(8082))
	promEntry.Bootstrap(context.Background())
	defer promEntry.Interrupt(context.Background())

	// 2: Create grpc client
	conn, client := createGreeterClient(opts...)
	defer conn.Close()

	// 3: Wrap context, this is required in order to use bellow features easily.
	ctx := rkgrpcctx.WrapContext(context.Background())

	// 4: Call server
	if resp, err := client.SayHello(ctx, &proto.HelloRequest{Name: "rk-dev"}); err != nil {
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to send request to server.", zap.Error(err))
	} else {
		rkgrpcctx.GetLogger(ctx).Info(fmt.Sprintf("[Message]: %s", resp.Message))
	}

	// 5: Wait for ctrl-C to shutdown server
	rkentry.GlobalAppCtx.WaitForShutdownSig()
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
