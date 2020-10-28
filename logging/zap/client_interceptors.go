// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_inter_logging

import (
	"github.com/rookie-ninja/rk-interceptor/context"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"path"
	"time"
	"unsafe"
)

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	// Merge option
	mergeOpt(opts)

	appName = defaultOptions.eventFactory.GetAppName()

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		event := defaultOptions.eventFactory.CreateEvent()
		event.SetStartTime(time.Now())

		// 1: Before invoking
		newCtx, callOpt := unaryClientBefore(ctx, method, cc, event)

		opts = append(opts, callOpt)

		// 2: Invoking
		err := invoker(newCtx, method, req, resp, cc, opts...)

		// 3: After invoking
		unaryClientAfter(newCtx, req, resp, err, method)

		return err
	}
}

func unaryClientBefore(ctx context.Context, method string, cc *grpc.ClientConn, event rk_query.Event) (context.Context, grpc.CallOption) {
	newCtx := recordClientBefore(ctx, event, method, "unary_client", cc)

	// Set headers for internal usage
	opt := grpc.Header(rk_inter_context.GetIncomingMD(newCtx))

	return metadata.NewOutgoingContext(newCtx, *rk_inter_context.GetOutgoingMD(newCtx)), opt
}

func unaryClientAfter(ctx context.Context, req, resp interface{}, err error, method string) {
	event := recordClientAfter(ctx, err, method)

	if defaultOptions.enableLogging && defaultOptions.enablePayloadLogging {
		event.AddFields(zap.String("request_payload", interfaceToString(req, maxRequestStrLen)),
			zap.String("response_payload", interfaceToString(resp, maxResponseStrLen)))
	}

	// Log to metrics if enabled
	if defaultOptions.enableMetrics {
		code := defaultOptions.errorToCode(err)
		method := path.Base(method)
		getClientBytesTransInMetrics(method, code.String()).Add(float64(unsafe.Sizeof(req)))
		getClientBytesTransOutMetrics(method, code.String()).Add(float64(unsafe.Sizeof(resp)))
	}

	event.WriteLog()
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	// Merge option
	mergeOpt(opts)

	appName = defaultOptions.eventFactory.GetAppName()

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		event := defaultOptions.eventFactory.CreateEvent()
		event.SetStartTime(time.Now())

		// 1: Before invoking
		newCtx := streamClientBefore(ctx, method, cc, event)

		// 2: Invoking
		clientStream, err := streamer(newCtx, desc, cc, method, opts...)

		// 3: After invoking
		streamClientAfter(newCtx, err, method)
		return clientStream, err
	}
}

func streamClientBefore(ctx context.Context, method string, cc *grpc.ClientConn, event rk_query.Event) context.Context {
	newCtx := recordClientBefore(ctx, event, method, "stream_client", cc)

	return metadata.NewOutgoingContext(newCtx, *rk_inter_context.GetOutgoingMD(newCtx))
}

func streamClientAfter(ctx context.Context, err error, method string) {
	event := recordClientAfter(ctx, err, method)
	event.WriteLog()
}

func recordClientBefore(ctx context.Context, event rk_query.Event, method, role string, cc *grpc.ClientConn) context.Context {
	outgoingRequestIds := rk_inter_context.GetRequestIdsFromOutgoingMD(ctx)

	remoteIP, remotePort, _ := net.SplitHostPort(cc.Target())
	event.SetRemoteAddr(remoteIP)
	event.SetOperation(path.Base(method))

	fields := []zap.Field{
		realm, region, az, domain, appVersion, localIP,
		zap.String("remote.IP", remoteIP),
		zap.String("remote.port", remotePort),
		zap.String("api.service", path.Dir(method)[1:]),
		zap.String("api.verb", path.Base(method)),
		zap.String("api.role", role),
		zap.Strings("outgoing_request_id", outgoingRequestIds),
		zap.Time("start_time", event.GetStartTime()),
	}

	defaultOptions.logger = defaultOptions.logger.With(zap.Strings("outgoing_request_id", outgoingRequestIds))

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	event.AddFields(fields...)

	// Extract outgoing metadata from context
	outgoingMD := rk_inter_context.GetOutgoingMD(ctx)
	incomingMD := rk_inter_context.GetIncomingMD(ctx)

	return rk_inter_context.ToContext(ctx, event, defaultOptions.logger, incomingMD, outgoingMD)
}

func recordClientAfter(ctx context.Context, err error, method string) rk_query.Event {
	code := defaultOptions.errorToCode(err)
	event := rk_inter_context.GetEvent(ctx)
	event.AddErr(err)
	endTime := time.Now()
	elapsed := endTime.Sub(event.GetStartTime())

	if defaultOptions.enableLogging {
		fields := make([]zap.Field, 0)

		// Check whether context is cancelled from server
		select {
		case <-ctx.Done():
			event.AddErr(ctx.Err())
			fields = append(fields, zap.NamedError("server_error", ctx.Err()))
		default:
			break
		}

		// extract request id and log it
		incomingRequestIds := rk_inter_context.GetRequestIdsFromIncomingMD(ctx)
		fields = append(fields,
			zap.String("res_code", code.String()),
			zap.Time("end_time", time.Now()),
			zap.Int64("elapsed_ms", elapsed.Nanoseconds()/1e6),
			zap.Strings("incoming_request_id", incomingRequestIds))

		rk_inter_context.SetLogger(ctx,
			rk_inter_context.GetLogger(ctx).With(
				zap.Strings("incoming_request_id", incomingRequestIds)))

		event.AddFields(fields...)
		if len(event.GetEventId()) < 1 {
			ids := append(rk_inter_context.GetRequestIdsFromIncomingMD(ctx), rk_inter_context.GetRequestIdsFromOutgoingMD(ctx)...)
			if len(ids) > 0 {
				event.SetEventId(interfaceToString(ids, 1000000))
			}
		}
		event.SetEndTime(endTime)
	}

	// Log to metrics if enabled
	if defaultOptions.enableMetrics {
		method := path.Base(method)
		getClientDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds() / 1e6))
		if err != nil {
			getClientErrorMetrics(method, code.String()).Inc()
		}
		getClientResCodeMetrics(method, code.String()).Inc()
	}

	return event
}
