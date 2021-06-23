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
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/tracing/telemetry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
)

// In this example, we will create a simple grpc client and enable trace interceptor.
func main() {
	// ****************************************
	// ********** Create Exporter *************
	// ****************************************

	// Export trace to stdout
	// exporter := rkgrpctrace.CreateFileExporter("stdout")

	// Export trace to local file system
	// exporter := rkgrpctrace.CreateFileExporter("logs/trace.log")

	// Export trace to jaeger collector
	// exporter := rkgrpctrace.CreateJaegerExporter("localhost:14268", "", "")

	// ********************************************
	// ********** Enable interceptors *************
	// ********************************************
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			rkgrpclog.UnaryClientInterceptor(),
			rkgrpctrace.UnaryClientInterceptor(
			// Entry name and entry type will be used for distinguishing interceptors. Recommended.
			// rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
			//
			// Provide an exporter.
			// rkgrpctrace.WithExporter(exporter),
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

	// 4: Wait for ctrl-C to shutdown server
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
