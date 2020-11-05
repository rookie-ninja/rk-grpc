// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"encoding/json"
	"github.com/rookie-ninja/rk-grpc/example/proto"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/retry"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {
	// create event factory
	factory := rk_query.NewEventFactory()

	// create client interceptor
	opt := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			rk_grpc_log.UnaryClientInterceptor(
				rk_grpc_log.WithEventFactory(factory),
				rk_grpc_log.WithLogger(rk_logger.StdoutLogger)),
			rk_grpc_retry.UnaryClientInterceptor()),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// Set up a connection to the server.
	conn, err := grpc.DialContext(context.Background(), "localhost:8080", opt...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// create grpc client
	c := proto.NewGreeterClient(conn)
	// create with rk context
	ctx, cancel := context.WithTimeout(rk_grpc_ctx.NewContext(), 5*time.Second)
	defer cancel()

	// add metadata
	rk_grpc_ctx.AddToOutgoingMD(ctx, "key", "1", "2")
	// add request id
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	// call server
	r, err := c.SayHello(ctx, &proto.HelloRequest{Name: "name"})

	rk_grpc_ctx.GetLogger(ctx).Info("This is info message")

	// print incoming metadata
	bytes, _ := json.Marshal(rk_grpc_ctx.GetIncomingMD(ctx))
	println(string(bytes))

	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
