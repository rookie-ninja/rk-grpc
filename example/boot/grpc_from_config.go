// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"github.com/rookie-ninja/rk-grpc/boot"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	"time"
)

func main() {
	fac := rk_query.NewEventFactory()
	entry := rk_grpc.NewGRpcEntries("example/boot/boot.yaml", fac, rk_logger.StdoutLogger)
	entry["greeter"].Bootstrap(fac.CreateEvent())
	entry["greeter"].Wait(1 * time.Second)
}
