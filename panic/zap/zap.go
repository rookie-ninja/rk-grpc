package zap

import (
	"fmt"
	"git.code.oa.com/pulse-line/pl_interceptor/logging/zap/ctxzap"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"runtime"
)

func LogPanicWithStackTrace(ctx context.Context, r interface{}) {
	logger := ctxzap.ExtractQueryLogger(ctx)

	logger.Error("recovered from panic",
		zap.String("panic", fmt.Sprintf("%v", r)),
		zapStack(),
	)
}

func zapStack() zap.Field {
	callers := []string{}
	for i := 0; true; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		callers = append(callers, fmt.Sprintf("%s(%d): %s", file, line, fn.Name()))
	}
	return zap.Any("stacktrace", callers)
}
