// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rookie-ninja/rk-grpc/example/interceptor/proto"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
	"github.com/rookie-ninja/rk-grpc/interceptor/panic"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
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

	// create event factory
	factory := rk_query.NewEventFactory()

	// create server interceptor
	opt := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			rk_grpc_log.UnaryServerInterceptor(
				rk_grpc_log.WithEventFactory(factory),
				rk_grpc_log.WithLogger(rk_logger.StdoutLogger)),
			rk_grpc_panic.UnaryServerInterceptor(rk_grpc_panic.PanicToStderr)),
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
	event := rk_grpc_ctx.GetEvent(ctx)
	// add fields
	event.AddFields(zap.String("key", "value"))
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
	rk_grpc_ctx.AddToOutgoingMD(ctx, "key", "1", "2")
	// add request id
	rk_grpc_ctx.AddRequestIdToOutgoingMD(ctx)

	// print incoming metadata
	bytes, _ := json.Marshal(rk_grpc_ctx.GetIncomingMD(ctx))
	println(string(bytes))

	rk_grpc_ctx.GetLogger(ctx).Info("this is info message")

	return &proto.HelloResponse{
		Message: "hello",
	}, nil
}
