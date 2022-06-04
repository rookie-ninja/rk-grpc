// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"embed"
	_ "embed"
	"github.com/rookie-ninja/rk-entry/v2/entry"
	"github.com/rookie-ninja/rk-grpc/v2/boot"
	proto "github.com/rookie-ninja/rk-grpc/v2/example/boot/simple/api/gen/v1"
	"google.golang.org/grpc"
)

//go:embed boot.yaml
var boot []byte

//go:embed api/gen/v1
var docsFS embed.FS

//go:embed api/gen/v1
var staticFS embed.FS

func init() {
	rkentry.GlobalAppCtx.AddEmbedFS(rkentry.DocsEntryType, "greeter", &docsFS)
	rkentry.GlobalAppCtx.AddEmbedFS(rkentry.SWEntryType, "greeter", &docsFS)
	rkentry.GlobalAppCtx.AddEmbedFS(rkentry.StaticFileHandlerEntryType, "greeter", &staticFS)
}

func main() {
	// Bootstrap basic entries from boot config.
	rkentry.BootstrapBuiltInEntryFromYAML(boot)
	rkentry.BootstrapPluginEntryFromYAML(boot)

	// Bootstrap grpc entry from boot config
	res := rkgrpc.RegisterGrpcEntryYAML(boot)

	// Get GrpcEntry
	grpcEntry := res["greeter"].(*rkgrpc.GrpcEntry)
	// Register gRPC server
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		proto.RegisterGreeterServer(server, &GreeterServer{})
	})
	// Register grpc-gateway func
	grpcEntry.AddRegFuncGw(proto.RegisterGreeterHandlerFromEndpoint)

	// Bootstrap grpc entry
	grpcEntry.Bootstrap(context.Background())

	// Wait for shutdown signal
	rkentry.GlobalAppCtx.WaitForShutdownSig()

	// Interrupt gin entry
	grpcEntry.Interrupt(context.Background())
}

// GreeterServer Implementation of GreeterServer.
type GreeterServer struct{}

// Greeter Handle Greeter method.
func (server *GreeterServer) Greeter(ctx context.Context, req *proto.GreeterRequest) (*proto.GreeterResponse, error) {
	return &proto.GreeterResponse{}, nil
}
