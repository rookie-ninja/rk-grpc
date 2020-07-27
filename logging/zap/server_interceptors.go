package rk_logging_zap

import (
	"github.com/rookie-ninja/rk-interceptor"
	"github.com/rookie-ninja/rk-interceptor/context"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"path"
	"time"
	"unsafe"
)

func UnaryServerInterceptor(queryLogger, appLogger *zap.Logger, opts ...Option) grpc.UnaryServerInterceptor {
	// Load server options provided
	// It will load default options if not provided
	opt := EvaluateServerOpt(opts)

	// Initiate event data factory at beginning with query logger
	createEventFactoryAsNeeded(queryLogger)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// 1: Before invoking
		newCtx := unaryServerBefore(ctx, queryLogger, appLogger, startTime, info)

		// 2: Invoking
		resp, err := handler(newCtx, req)

		// 3: After invoking
		unaryServerAfter(newCtx, req, resp, opt, err, info, startTime)

		return resp, err
	}
}

func unaryServerBefore(ctx context.Context, queryLogger, appLogger *zap.Logger, startTime time.Time, info *grpc.UnaryServerInfo) context.Context {
	// Add request ids from remote side
	clientRequestIds := rk_context.GetRequestIdsFromIncomingMD(ctx)
	serverRequestIds := rk_context.GetRequestIdsFromOutgoingMD(ctx)

	fields := []zap.Field{
		realmField,
		regionField,
		azField,
		domainField,
		appVersionField,
		appNameField,
		localIP,
		localHostname,
		zap.String("api.service", path.Dir(info.FullMethod)[1:]),
		zap.String("api.verb", path.Base(info.FullMethod)),
		zap.String("api.role", "unary_server"),
		zap.Time("start_time", startTime),
		zap.Strings("incoming_request_id", clientRequestIds),
		zap.Strings("outgoing_request_id", serverRequestIds),
	}

	fields = append(fields, getRemoteAddressSet(ctx)...)

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	event := eventFactory.CreateEvent()
	// Not a good way, but event data needs to start first
	// Or it won't be able to start any timer
	// It is OK not to end the event data since it won't be reused anymore
	event.SetStartTimeMS(startTime.Unix()/1000/1000)

	incomingMetadata := rk_context.GetIncomingMD(ctx)
	outgoingMetadata := rk_context.GetOutgoingMD(ctx)

	return rk_context.ToContext(ctx, queryLogger.With(fields...), appLogger, event, incomingMetadata, outgoingMetadata)
}

func unaryServerAfter(ctx context.Context, req, resp interface{}, opt *Options, err error, info *grpc.UnaryServerInfo, startTime time.Time) {
	endTime := time.Now()
	elapsed := endTime.Sub(startTime)

	code := opt.codeFunc(err)

	// Set response as header
	grpc.SetHeader(ctx, *rk_context.GetOutgoingMD(ctx))

	// Log to query logger if enabled
	if opt.enableLog(info.FullMethod, err) {
		fields := []zap.Field{
			zap.Error(err),
			zap.String("server_code", code.String()),
			zap.Time("end_time", endTime),
			zap.Int64("elapsed_ms", elapsed.Nanoseconds()/1000/1000),
		}

		if opt.enablePayload(info.FullMethod, err) {
			fields = append(fields,
				zap.String("request_payload", interfaceToString(req, maxRequestStrLen)),
				zap.String("response_payload", interfaceToString(resp, maxResponseStrLen)))
		}

		// extract event data and log it
		fields = append(fields, rk_context.GetEvent(ctx).ToZapFieldsMin()...)

		// Check whether context is cancelled from client
		select {
		case <-ctx.Done():
			fields = append(fields, zap.NamedError("client_error", ctx.Err()))
		default:
			break
		}

		// extract request id and log it
		clientRequestIds := rk_context.GetRequestIdsFromIncomingMD(ctx)
		fields = append(fields, zap.Strings("client_request_id", clientRequestIds))

		serverRequestIds := rk_context.GetRequestIdsFromOutgoingMD(ctx)
		fields = append(fields, zap.Strings("server_request_id", serverRequestIds))

		// re-extract logger from newCtx, as it may have extra fields that changed in the holder.
		rk_context.GetQueryLogger(ctx).Check(zap.InfoLevel, gRPCRequest).Write(fields...)
	}

	// Log to metrics if enabled
	if opt.enableProm(info.FullMethod, err) {
		method := path.Base(info.FullMethod)
		getServerDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds()/1000/1000))
		if err != nil {
			getServerErrorMetrics(method, code.String()).Inc()
		}
		getServerResCodeMetrics(method, code.String()).Inc()
		getServerBytesTransInMetrics(method, code.String()).Add(float64(unsafe.Sizeof(req)))
		getServerBytesTransOutMetrics(method, code.String()).Add(float64(unsafe.Sizeof(resp)))
	}
}

func StreamServerInterceptor(queryLogger *zap.Logger, appLogger *zap.Logger, opts ...Option) grpc.StreamServerInterceptor {
	// Load server options provided
	// It will load default options if not provided
	o := EvaluateServerOpt(opts)

	// Initiate event data factory at beginning with query logger
	createEventFactoryAsNeeded(queryLogger)

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		// 1: Before invoking
		wrappedStream := streamServerBefore(stream, queryLogger, appLogger, info, startTime)

		// 2: Invoking
		err := handler(srv, wrappedStream)

		// 3: After invoking
		streamServerAfter(wrappedStream, o, err, info, startTime)

		return err
	}
}

func streamServerBefore(stream grpc.ServerStream, queryLogger, appLogger *zap.Logger, info *grpc.StreamServerInfo, startTime time.Time) grpc.ServerStream {
	ctx := stream.Context()

	fields := []zap.Field{
		realmField,
		regionField,
		azField,
		domainField,
		appVersionField,
		appNameField,
		localIP,
		localHostname,
		zap.String("api.service", path.Dir(info.FullMethod)[1:]),
		zap.String("api.verb", path.Base(info.FullMethod)),
		zap.String("api.role", "stream_server"),
		zap.Time("start_time", startTime),
	}

	fields = append(fields, getRemoteAddressSet(ctx)...)

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	eventData := eventFactory.CreateEvent()
	// Not a good way, but event data needs to start first
	// Or it won't be able to start any timer
	// It is OK not to end the event data since it won't be reused anymore
	eventData.SetStartTimeMS(startTime.Unix())

	incomingMetadata := rk_context.GetIncomingMD(ctx)
	outgoingMetadata := rk_context.GetOutgoingMD(ctx)

	newCtx := rk_context.ToContext(ctx, queryLogger.With(fields...), appLogger, eventData, incomingMetadata, outgoingMetadata)

	wrappedStream := rk_interceptor.WrapServerStream(stream)
	wrappedStream.WrappedContext = newCtx

	return wrappedStream
}

func streamServerAfter(stream grpc.ServerStream, o *Options, err error, info *grpc.StreamServerInfo, startTime time.Time) {
	endTime := time.Now()
	ctx := stream.Context()

	elapsed := endTime.Sub(startTime)

	code := o.codeFunc(err)

	// Log to query logger if enabled
	if o.enableLog(info.FullMethod, err) {
		fields := []zap.Field{
			zap.Error(err),
			zap.String("res_code", code.String()),
			zap.Time("end_time", endTime),
			zap.Int64("elapsed_ms", elapsed.Nanoseconds()/1000/1000),
		}

		fields = append(fields, rk_context.GetEvent(ctx).ToZapFieldsMin()...)

		// Check whether context is cancelled from client
		select {
		case <-ctx.Done():
			fields = append(fields, zap.NamedError("client_error", ctx.Err()))
		default:
			break
		}

		// extract request id and log it
		clientRequestIds := rk_context.GetRequestIdsFromIncomingMD(ctx)
		fields = append(fields, zap.Strings("client_request_id", clientRequestIds))

		serverRequestIds := rk_context.GetRequestIdsFromOutgoingMD(ctx)
		fields = append(fields, zap.Strings("server_request_id", serverRequestIds))

		// re-extract logger from newCtx, as it may have extra fields that changed in the holder.
		rk_context.GetQueryLogger(ctx).Check(zap.InfoLevel, gRPCRequest).Write(fields...)
	}

	// Log to metrics if enabled
	if o.enableProm(info.FullMethod, err) {
		method := path.Base(info.FullMethod)
		getServerDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds() / 1000 / 1000))
		if err != nil {
			getServerErrorMetrics(method, code.String()).Inc()
		}
		getServerResCodeMetrics(method, code.String()).Inc()
	}
}
