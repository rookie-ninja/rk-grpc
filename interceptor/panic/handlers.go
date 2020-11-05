// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc_panic

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.uber.org/zap"
	"os"
	"runtime"
)

type PanicHandler func(context.Context, interface{})

var handlers []PanicHandler

func PanicToStderr(ctx context.Context, r interface{}) {
	os.Stderr.WriteString(panicString(ctx, r))
}

func PanicToGLog(ctx context.Context, r interface{}) {
	glog.Error(panicString(ctx, r))
}

func PanicToZap(ctx context.Context, r interface{}) {
	rk_grpc_ctx.GetEvent(ctx).AddErr(panic{})
	rk_grpc_ctx.GetEvent(ctx).AddFields(zap.String("stacktrace", panicString(ctx, r)))
}

func panicString(ctx context.Context, r interface{}) string {
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("\npanic: runtime error by %v\n", r))
	for i := 0; true; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		buffer.WriteString(fmt.Sprintf("\tat %s (%s:%d)\n", fn.Name(), file, line))
	}

	return buffer.String()
}
