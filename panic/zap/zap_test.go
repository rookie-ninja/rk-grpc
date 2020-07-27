package zap

import (
	"context"
	"git.code.oa.com/pulse-line/pl_interceptor/logging/zap/ctxzap"
	"git.code.oa.com/pulse-line/pl_interceptor/panichandler"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestUnaryServer(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "TestService.UnaryMethod",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test error")
	}

	panichandler.InstallPanicHandler(LogPanicWithStackTrace)

	ctx := context.Background()
	ctx = ctxzap.NewPlContext(logger, logger)
	_, err := panichandler.UnaryServerInterceptor(ctx, "xyz", unaryInfo, unaryHandler)
	if err == nil {
		t.Fatalf("unexpected success")
	}

	if got, want := status.Code(err), codes.Internal; got != want {
		t.Errorf("expect grpc.Code to %s, but got %s", want, got)
	}
}
