// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-grpc/boot"
	"net/http"
)

func main() {
	// Bootstrap basic entries from boot config.
	rkentry.RegisterInternalEntriesFromConfig("example/boot/cors/boot.yaml")

	// Bootstrap grpc entry from boot config
	res := rkgrpc.RegisterGrpcEntriesWithConfig("example/boot/cors/boot.yaml")

	// Bootstrap grpc entry
	res["greeter"].Bootstrap(context.Background())

	entry := res["greeter"].(*rkgrpc.GrpcEntry)
	entry.GwMux.HandlePath("GET", "/rk/v1/greeter", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.Write([]byte(fmt.Sprintf("CSRF token:%s", r.Header.Get("_csrf"))))
	})

	entry.GwMux.HandlePath("POST", "/rk/v1/greeter", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.Write([]byte(fmt.Sprintf("CSRF token:%s", r.Header.Get("_csrf"))))
	})

	// Wait for shutdown signal
	rkentry.GlobalAppCtx.WaitForShutdownSig()

	// Interrupt grpc entry
	res["greeter"].Interrupt(context.Background())
}
