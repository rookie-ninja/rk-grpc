package rk_grpc

import (
	"github.com/golang/glog"
	"os"
	"runtime/debug"
)

func shutdownWithError(err error) {
	debug.PrintStack()
	glog.Error(err)
	os.Exit(1)
}

