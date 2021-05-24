// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/example/interceptor/proto"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"google.golang.org/grpc"
	"log"
	"time"
)

func main() {
	entryName := "example-entry-client-name"
	entryType := "example-entry-client"

	// create server interceptor
	basicInter := rkgrpcbasic.UnaryClientInterceptor(
		rkgrpcbasic.WithEntryNameAndType(entryName, entryType))

	logInter := rkgrpclog.UnaryClientInterceptor(
		rkgrpclog.WithEntryNameAndType(entryName, entryType),
		rkgrpclog.WithZapLoggerEntry(rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()),
		rkgrpclog.WithEventLoggerEntry(rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()))

	metricsInter := rkgrpcmetrics.UnaryClientInterceptor(
		rkgrpcmetrics.WithEntryNameAndType(entryName, entryType),
		rkgrpcmetrics.WithRegisterer(prometheus.NewRegistry()))

	panicInter := rkgrpcpanic.UnaryClientInterceptor()

	// create client interceptor
	opt := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			basicInter,
			logInter,
			metricsInter,
			panicInter,
		),
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
	ctx, cancel := context.WithTimeout(rkgrpcctx.NewContext(), 5*time.Second)
	defer cancel()

	// add metadata
	rkgrpcctx.AddToOutgoingMD(ctx, "key", "1", "2")
	// add request id
	rkgrpcctx.AddRequestIdToOutgoingMD(ctx)

	// call server
	r, err := c.SayHello(ctx, &proto.HelloRequest{Name: "name"})

	rkgrpcctx.GetZapLogger(ctx).Info("This is info message")

	// print incoming metadata
	bytes, _ := json.Marshal(rkgrpcctx.GetIncomingMD(ctx))
	println(string(bytes))

	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
