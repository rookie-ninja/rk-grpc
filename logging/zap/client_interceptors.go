package rk_logging_zap

import (
	"github.com/rookie-ninja/rk-interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"path"
	"time"
	"unsafe"
)

func UnaryClientInterceptor(queryLogger *zap.Logger, appLogger *zap.Logger, opts ...Option) grpc.UnaryClientInterceptor {
	o := EvaluateClientOpt(opts)

	// Initiate event data factory at beginning with query logger
	createEventFactoryAsNeeded(queryLogger)

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		startTime := time.Now()

		// 1: Before invoking
		newCtx, opt := unaryClientBefore(ctx, method, cc, queryLogger, appLogger, startTime)

		opts = append(opts, opt)

		// 2: Invoking
		err := invoker(newCtx, method, req, resp, cc, opts...)

		// 3: After invoking
		unaryClientAfter(newCtx, req, resp, o, err, method, startTime)

		return err
	}
}

func unaryClientBefore(ctx context.Context, method string, cc *grpc.ClientConn, queryLogger, appLogger *zap.Logger, startTime time.Time) (context.Context, grpc.CallOption) {
	remoteIP, remotePort, _ := net.SplitHostPort(cc.Target())

	fields := []zap.Field{
		realmField,
		regionField,
		azField,
		domainField,
		appVersionField,
		appNameField,
		localIP,
		localHostname,
		zap.String("remote.IP", remoteIP),
		zap.String("remote.port", remotePort),
		zap.String("api.service", path.Dir(method)[1:]),
		zap.String("api.verb", path.Base(method)),
		zap.String("api.role", "unary_client"),
		zap.Time("start_time", startTime),
	}

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	// Extract outgoing metadata from context
	outgoingMD := rk_context.GetOutgoingMD(ctx)
	incomingMD := rk_context.GetIncomingMD(ctx)

	plContext := rk_context.ToContext(ctx, queryLogger.With(fields...), appLogger, nil, incomingMD, outgoingMD)

	// Set headers for internal usage
	opt := grpc.Header(incomingMD)

	return metadata.NewOutgoingContext(plContext, *rk_context.GetOutgoingMD(plContext)), opt
}

func unaryClientAfter(ctx context.Context, req, resp interface{}, o *Options, err error, method string, startTime time.Time) {
	endTime := time.Now()
	elapsed := endTime.Sub(startTime)

	code := o.codeFunc(err)

	if o.enableLog(method, err) {
		fields := []zap.Field{
			zap.Error(err),
			zap.String("server_code", code.String()),
			zap.Time("end_time", time.Now()),
			zap.Int64("elapsed_ms", elapsed.Nanoseconds()/1000/1000),
		}

		if o.enablePayload(method, err) {
			fields = append(fields,
				zap.String("request_payload", interfaceToString(req, maxRequestStrLen)),
				zap.String("response_payload", interfaceToString(resp, maxResponseStrLen)))
		}

		// extract request id and log it
		serverRequestIds := rk_context.GetRequestIdsFromIncomingMD(ctx)
		fields = append(fields, zap.Strings("server_request_id", serverRequestIds))

		clientRequestIds := rk_context.GetRequestIdsFromOutgoingMD(ctx)
		fields = append(fields, zap.Strings("client_request_id", clientRequestIds))

		rk_context.GetQueryLogger(ctx).Check(zap.InfoLevel, gRPCRequest).Write(fields...)
	}

	// Log to metrics if enabled
	if o.enableProm(method, err) {
		method := path.Base(method)
		getServerDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds() / 1000 / 1000))
		if err != nil {
			getClientErrorMetrics(method, code.String()).Inc()
		}
		getClientResCodeMetrics(method, code.String()).Inc()
		getClientBytesTransInMetrics(method, code.String()).Add(float64(unsafe.Sizeof(req)))
		getClientBytesTransOutMetrics(method, code.String()).Add(float64(unsafe.Sizeof(resp)))
	}
}

func StreamClientInterceptor(queryLogger *zap.Logger, appLogger *zap.Logger, opts ...Option) grpc.StreamClientInterceptor {
	opt := EvaluateClientOpt(opts)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		startTime := time.Now()
		// 1: Before invoking
		fields, newCtx := streamClientBefore(ctx, method, cc, queryLogger, appLogger, startTime)

		// 2: Invoking
		clientStream, err := streamer(newCtx, desc, cc, method, opts...)

		// 3: After invoking
		streamClientAfter(opt, err, queryLogger.With(fields...), method, startTime)
		return clientStream, err
	}
}

func streamClientBefore(ctx context.Context, method string, cc *grpc.ClientConn, queryLogger, appLogger *zap.Logger, startTime time.Time) ([]zap.Field, context.Context) {
	remoteIP, remotePort, _ := net.SplitHostPort(cc.Target())

	fields := []zap.Field{
		realmField,
		regionField,
		azField,
		domainField,
		appVersionField,
		appNameField,
		localIP,
		localHostname,
		zap.String("remote.IP", remoteIP),
		zap.String("remote.port", remotePort),
		zap.String("api.service", path.Dir(method)[1:]),
		zap.String("api.verb", path.Base(method)),
		zap.String("api.role", "stream_client"),
		zap.Time("start_time", startTime),
	}

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	// Extract outgoing metadata from context
	outgoingMD := rk_context.GetOutgoingMD(ctx)
	incomingMD := rk_context.GetIncomingMD(ctx)

	newCtx := rk_context.ToContext(ctx, queryLogger.With(fields...), appLogger, nil, incomingMD, outgoingMD)

	return fields, metadata.NewOutgoingContext(newCtx, *rk_context.GetOutgoingMD(newCtx))
}

func streamClientAfter(opt *Options, err error, logger *zap.Logger, method string, startTime time.Time) {
	code := opt.codeFunc(err)

	endTime := time.Now()
	elapsed := endTime.Sub(startTime)

	if opt.enableLog(method, err) {
		logger.Check(zap.InfoLevel, gRPCRequest).Write(
			zap.Error(err),
			zap.String("server_code", code.String()),
			zap.Time("end_time", time.Now()),
			zap.Int64("elapsed_ms", elapsed.Nanoseconds()/1000/1000))
	}

	// Log to metrics if enabled
	if opt.enableProm(method, err) {
		method := path.Base(method)
		getClientDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds() / 1000 / 1000))
		if err != nil {
			getClientErrorMetrics(method, code.String()).Inc()
		}
		getClientResCodeMetrics(method, code.String()).Inc()
	}

}