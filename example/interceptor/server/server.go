// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/example/interceptor/proto"
	"github.com/rookie-ninja/rk-grpc/interceptor/auth/basic_auth"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/metrics/prom"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

func main() {
	// create listener
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	entryName := "example-entry-server-name"
	entryType := "example-entry-client"

	// create server interceptor
	// basic interceptor
	basicInter := rkgrpcbasic.UnaryServerInterceptor(
		rkgrpcbasic.WithEntryNameAndType(entryName, entryType))

	// logging interceptor
	logInter := rkgrpclog.UnaryServerInterceptor(
		rkgrpclog.WithEntryNameAndType(entryName, entryType),
		rkgrpclog.WithZapLoggerEntry(rkentry.GlobalAppCtx.GetZapLoggerEntryDefault()),
		rkgrpclog.WithEventLoggerEntry(rkentry.GlobalAppCtx.GetEventLoggerEntryDefault()))

	// prometheus metrics interceptor
	metricsInter := rkgrpcmetrics.UnaryServerInterceptor(
		rkgrpcmetrics.WithEntryNameAndType(entryName, entryType),
		rkgrpcmetrics.WithRegisterer(prometheus.NewRegistry()))

	// basic auth interceptor
	basicAuthInter := rkgrpcbasicauth.UnaryServerInterceptor(
		rkgrpcbasicauth.WithEntryNameAndType(entryName, entryType),
		rkgrpcbasicauth.WithCredential("user:name"))

	// panic interceptor
	panicInter := rkgrpcpanic.UnaryServerInterceptor()

	opt := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			basicInter,
			logInter,
			metricsInter,
			basicAuthInter,
			panicInter),
	}

	// create server
	s := grpc.NewServer(opt...)
	proto.RegisterGreeterServer(s, &GreeterServer{})

	// serving
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type GreeterServer struct{}

func (server *GreeterServer) SayHello(ctx context.Context, request *proto.HelloRequest) (*proto.HelloResponse, error) {
	event := rkgrpcctx.GetEvent(ctx)
	// add fields
	event.AddPayloads(zap.String("key", "value"))
	// add error
	event.AddErr(errors.New(""))
	// add pair
	event.AddPair("key", "value")
	// set counter
	event.SetCounter("ctr", 1)
	// timer
	event.StartTimer("sleep")
	time.Sleep(1 * time.Second)
	event.EndTimer("sleep")
	// add to metadata
	rkgrpcctx.AddToOutgoingMD(ctx, "key", "1", "2")
	// add request id
	rkgrpcctx.SetRequestIdToOutgoingMD(ctx)

	// print incoming metadata
	bytes, _ := json.Marshal(rkgrpcctx.GetIncomingMD(ctx))
	println(string(bytes))

	// print with logger to check whether id was printed
	rkgrpcctx.GetLogger(ctx).Info("this is info message")

	return &proto.HelloResponse{
		Message: "hello",
	}, nil
}
