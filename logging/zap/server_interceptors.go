// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_logging_zap

import (
	"github.com/rookie-ninja/rk-interceptor"
	"github.com/rookie-ninja/rk-interceptor/context"
	rk_query "github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"path"
	"time"
	"unsafe"
)

func UnaryServerInterceptor(factory *rk_query.EventFactory, opts ...Option) grpc.UnaryServerInterceptor {
	// Merge option
	opt := MergeOpt(opts)

	// We will populate Noop Zap logger if factory is nil
	if factory == nil {
		factory = rk_query.NewEventFactory()
	}

	eventFactory = factory
	appName = factory.GetAppName()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		event := eventFactory.CreateEvent()
		event.SetStartTime(time.Now())

		// 1: Before invoking
		newCtx := unaryServerBefore(ctx, event, info)

		// 2: Invoking
		resp, err := handler(newCtx, req)

		// 3: After invoking
		unaryServerAfter(newCtx, req, resp, opt, err, info)

		return resp, err
	}
}

func unaryServerBefore(ctx context.Context, event rk_query.Event, info *grpc.UnaryServerInfo) context.Context {
	return recordServerBefore(ctx, event, info.FullMethod, "unary_server")
}

func unaryServerAfter(ctx context.Context, req, resp interface{}, opt *Options, err error, info *grpc.UnaryServerInfo) {
	event := recordServerAfter(ctx, opt, err, info.FullMethod)
	grpc.SetHeader(ctx, *rk_context.GetOutgoingMD(ctx))

	if opt.enableLogging(info.FullMethod, err) && opt.enablePayloadLogging(info.FullMethod, err) {
		event.AddFields(
			zap.String("request_payload", interfaceToString(req, maxRequestStrLen)),
			zap.String("response_payload", interfaceToString(resp, maxResponseStrLen)))
	}

	// Log to metrics if enabled
	if opt.enableMetrics(info.FullMethod, err) {
		code := opt.errorToCode(err)
		getServerBytesTransInMetrics(info.FullMethod, code.String()).Add(float64(unsafe.Sizeof(req)))
		getServerBytesTransOutMetrics(info.FullMethod, code.String()).Add(float64(unsafe.Sizeof(resp)))
	}
	event.WriteLog()
}

func StreamServerInterceptor(factory *rk_query.EventFactory, opts ...Option) grpc.StreamServerInterceptor {
	// Merge option
	opt := MergeOpt(opts)

	// We will populate Noop Zap logger if factory is nil
	if factory == nil {
		factory = rk_query.NewEventFactory()
	}

	eventFactory = factory
	appName = factory.GetAppName()

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		event := eventFactory.CreateEvent()
		event.SetStartTime(time.Now())

		// 1: Before invoking
		wrappedStream := streamServerBefore(stream, event, info)

		// 2: Invoking
		err := handler(srv, wrappedStream)

		// 3: After invoking
		streamServerAfter(wrappedStream, opt, err, info)

		return err
	}
}

func streamServerBefore(stream grpc.ServerStream, event rk_query.Event, info *grpc.StreamServerInfo) grpc.ServerStream {
	wrappedStream := rk_interceptor.WrapServerStream(stream)
	wrappedStream.WrappedContext = recordServerBefore(stream.Context(), event, info.FullMethod, "stream_server")

	return wrappedStream
}

func streamServerAfter(stream grpc.ServerStream, opt *Options, err error, info *grpc.StreamServerInfo) {
	event := recordServerAfter(stream.Context(), opt, err, info.FullMethod)
	event.WriteLog()
}

func recordServerBefore(ctx context.Context, event rk_query.Event, method, role string) context.Context {
	// Add request ids from remote side
	incomingRequestIds := rk_context.GetRequestIdsFromIncomingMD(ctx)

	fields := []zap.Field{
		realm, region, az, domain, appVersion, localIP,
		zap.String("api.service", path.Dir(method)[1:]),
		zap.String("api.verb", path.Base(method)),
		zap.String("api.role", role),
		zap.Strings("incoming_request_id", incomingRequestIds),
		zap.Time("start_time", event.GetStartTime()),
	}

	remoteAddressSet := getRemoteAddressSet(ctx)
	fields = append(fields, remoteAddressSet...)
	event.SetRemoteAddr(remoteAddressSet[0].String)
	event.SetOperation(path.Base(method))

	if d, ok := ctx.Deadline(); ok {
		event.AddErr(ctx.Err())
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	incomingMD := rk_context.GetIncomingMD(ctx)
	outgoingMD := rk_context.GetOutgoingMD(ctx)

	return rk_context.ToContext(ctx, event, incomingMD, outgoingMD, fields)
}

func recordServerAfter(ctx context.Context, opt *Options, err error, method string) rk_query.Event {
	code := opt.errorToCode(err)
	event := rk_context.GetEvent(ctx)
	event.AddErr(err)
	endTime := time.Now()
	elapsed := endTime.Sub(event.GetStartTime())

	// Log to query logger if enabled
	if opt.enableLogging(method, err) {
		fields := rk_context.GetFields(ctx)

		// Check whether context is cancelled from client
		select {
		case <-ctx.Done():
			event.AddErr(ctx.Err())
			fields = append(fields, zap.NamedError("client_error", ctx.Err()))
		default:
			break
		}

		// extract request id and log it
		outgoingRequestIds := rk_context.GetRequestIdsFromOutgoingMD(ctx)
		fields = append(fields,
			zap.String("res_code", code.String()),
			zap.Time("end_time", endTime),
			zap.Int64("elapsed_ms", elapsed.Milliseconds()),
			zap.Strings("outgoing_request_id", outgoingRequestIds),
		)

		event.AddFields(fields...)
		if len(event.GetEventId()) < 1 {
			ids := append(rk_context.GetRequestIdsFromIncomingMD(ctx), rk_context.GetRequestIdsFromOutgoingMD(ctx)...)
			if len(ids) > 0 {
				event.SetEventId(interfaceToString(ids, 1000000))
			}
		}
		event.SetEndTime(endTime)
	}

	// Log to metrics if enabled
	if opt.enableMetrics(method, err) {
		method := path.Base(method)
		getServerDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds() / 1e6))
		if err != nil {
			getServerErrorMetrics(method, code.String()).Inc()
		}
		getServerResCodeMetrics(method, code.String()).Inc()
	}

	return event
}
