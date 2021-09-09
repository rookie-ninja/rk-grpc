// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	proto "github.com/rookie-ninja/rk-grpc/example/interceptor/proto/testdata"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
)

// In this example, we will create a simple grpc client and enable log interceptor.
func main() {
	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			rkgrpclog.UnaryClientInterceptor(
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
	conn, client := createGreeterClient(opts...)
	defer conn.Close()

	// 2: Recommended: Wrap context, this is recommended.
	ctx := rkgrpcctx.WrapContext(context.Background())

	// *********************************************************
	// ********** Add headers will send to server **************
	// *********************************************************
	//
	// Send headers to server with this function
	//
	// rkgrpcctx.AddHeaderToServer(ctx, "from-client", "value")

	// 3: Call server
	if resp, err := client.SayHello(ctx, &proto.HelloRequest{Name: "rk-dev"}); err != nil {
		// Get call-scoped logger
		rkgrpcctx.GetLogger(ctx).Fatal("Failed to greet.", zap.Error(err))
	} else {
		// Get call-scoped logger
		rkgrpcctx.GetLogger(ctx).Info(fmt.Sprintf("[Message]: %s", resp.Message))
	}

	// **********************************************
	// ********** context-scoped logger *************
	// **********************************************
	//
	// RequestId will be printed if server returns with header of rkgrpcctx.RequestIdKey
	//
	// For client, the logger is context-scoped which means even if RPC finished, logger will
	// still print requestId and traceId if exists.
	//
	// rkgrpcctx.GetLogger(ctx).Info("Request sent to server.")

	// *********************************************
	// ********** Get incoming headers *************
	// *********************************************
	//
	// Read headers sent from server.
	//
	// for k, v := range rkgrpcctx.GetIncomingHeaders(ctx) {
	// 	 fmt.Println(fmt.Sprintf("%s: %s", k, v))
	// }
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
