package rkgrpctrace

import (
	"context"
	"github.com/rookie-ninja/rk-grpc/interceptor/basic"
	"github.com/rookie-ninja/rk-grpc/interceptor/context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor suitable
// for use in a grpc.Dial call.
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeUnaryClient, opts...)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 1: Before invoking
		ctx, span := clientBefore(ctx, set)

		// Set headers for internal usage
		md := rkgrpcctx.GetIncomingMD(ctx)
		opts = append(opts, grpc.Header(&md))

		// 2: Invoking
		err := invoker(ctx, method, req, resp, cc, opts...)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// 3: After invoking
		clientAfter(span, err)

		return err
	}
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	set := newOptionSet(rkgrpcbasic.RpcTypeStreamClient, opts...)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Before invoking
		ctx, span := clientBefore(ctx, set)

		// Set headers for internal usage
		md := rkgrpcctx.GetIncomingMD(ctx)
		opts = append(opts, grpc.Header(&md))

		// Invoking
		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		rkgrpcctx.GetRpcInfo(ctx).Err = err

		// After invoking
		clientAfter(span, err)

		return clientStream, err
	}
}

func clientBefore(ctx context.Context, set *optionSet) (context.Context, oteltrace.Span) {
	opts := []oteltrace.SpanOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(localeToAttributes()...),
		oteltrace.WithAttributes(grpcInfoToAttributes(ctx)...),
	}

	rpcInfo := rkgrpcctx.GetRpcInfo(ctx)

	// create span name
	spanName := rpcInfo.GrpcMethod
	if len(spanName) < 1 {
		spanName = "rk-span-default"
	}

	// create span
	ctx, span := set.Tracer.Start(ctx, spanName, opts...)

	// insert into context
	ctx = context.WithValue(ctx, "rk-trace-id", span.SpanContext().TraceID().String())
	rkgrpcctx.GetEvent(ctx).SetTraceId(span.SpanContext().TraceID().String())

	// inject into metadata
	outgoingMD, _ := metadata.FromOutgoingContext(ctx)
	outgoingMDCopy := outgoingMD.Copy()
	set.Propagator.Inject(ctx, &GrpcMetadataCarrier{md: &outgoingMDCopy})
	ctx = metadata.NewOutgoingContext(ctx, outgoingMDCopy)

	// return new context with tracer and traceId
	return rkgrpcctx.ToRkContext(ctx,
		rkgrpcctx.WithTracer(set.Tracer),
		rkgrpcctx.WithPropagator(set.Propagator),
		rkgrpcctx.WithTraceProvider(set.Provider)), span
}

func clientAfter(span oteltrace.Span, err error) {
	defer span.End()
	if err != nil {
		s, _ := status.FromError(err)
		span.SetStatus(codes.Error, s.Message())
		span.SetAttributes(attribute.Int("grpc.code", int(s.Code())))
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Int("grpc.code", int(codes.Ok)))
	}
}
