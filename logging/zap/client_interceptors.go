// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_logging_zap

import (
	"github.com/rookie-ninja/rk-interceptor/context"
	rk_query "github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"path"
	"time"
	"unsafe"
)

func UnaryClientInterceptor(factory *rk_query.EventFactory, opts ...Option) grpc.UnaryClientInterceptor {
	// Merge option
	opt := MergeOpt(opts)

	// We will populate Noop Zap logger if factory is nil
	if factory == nil {
		factory = rk_query.NewEventFactory()
	}

	eventFactory = factory
	appName = factory.GetAppName()

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		event := eventFactory.CreateEvent()
		event.SetStartTime(time.Now())

		// 1: Before invoking
		newCtx, callOpt := unaryClientBefore(ctx, method, cc, event)

		opts = append(opts, callOpt)

		// 2: Invoking
		err := invoker(newCtx, method, req, resp, cc, opts...)

		// 3: After invoking
		unaryClientAfter(newCtx, req, resp, opt, err, method)

		return err
	}
}

func unaryClientBefore(ctx context.Context, method string, cc *grpc.ClientConn, event rk_query.Event) (context.Context, grpc.CallOption) {
	newCtx := recordClientBefore(ctx, event, method, "unary_client", cc)

	// Set headers for internal usage
	opt := grpc.Header(rk_context.GetIncomingMD(newCtx))

	return metadata.NewOutgoingContext(newCtx, *rk_context.GetOutgoingMD(newCtx)), opt
}

func unaryClientAfter(ctx context.Context, req, resp interface{}, opt *Options, err error, method string) {
	event := recordClientAfter(ctx, opt, err, method)

	if opt.enableLogging(method, err) && opt.enablePayloadLogging(method, err) {
		event.AddFields(zap.String("request_payload", interfaceToString(req, maxRequestStrLen)),
			zap.String("response_payload", interfaceToString(resp, maxResponseStrLen)))
	}

	// Log to metrics if enabled
	if opt.enableMetrics(method, err) {
		code := opt.errorToCode(err)
		method := path.Base(method)
		getClientBytesTransInMetrics(method, code.String()).Add(float64(unsafe.Sizeof(req)))
		getClientBytesTransOutMetrics(method, code.String()).Add(float64(unsafe.Sizeof(resp)))
	}

	event.WriteLog()
}

func StreamClientInterceptor(factory *rk_query.EventFactory, opts ...Option) grpc.StreamClientInterceptor {
	// Merge option
	opt := MergeOpt(opts)

	// We will populate Noop Zap logger if factory is nil
	if factory == nil {
		factory = rk_query.NewEventFactory()
	}

	eventFactory = factory
	appName = factory.GetAppName()

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		event := eventFactory.CreateEvent()
		event.SetStartTime(time.Now())

		// 1: Before invoking
		newCtx := streamClientBefore(ctx, method, cc, event)

		// 2: Invoking
		clientStream, err := streamer(newCtx, desc, cc, method, opts...)

		// 3: After invoking
		streamClientAfter(newCtx, opt, err, method)
		return clientStream, err
	}
}

func streamClientBefore(ctx context.Context, method string, cc *grpc.ClientConn, event rk_query.Event) context.Context {
	newCtx := recordClientBefore(ctx, event, method, "stream_client", cc)

	return metadata.NewOutgoingContext(newCtx, *rk_context.GetOutgoingMD(newCtx))
}

func streamClientAfter(ctx context.Context, opt *Options, err error, method string) {
	event := recordClientAfter(ctx, opt, err, method)
	event.WriteLog()
}

func recordClientBefore(ctx context.Context, event rk_query.Event, method, role string, cc *grpc.ClientConn) context.Context {
	outgoingRequestIds := rk_context.GetRequestIdsFromOutgoingMD(ctx)

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

	if d, ok := ctx.Deadline(); ok {
		fields = append(fields, zap.String("deadline", d.Format(time.RFC3339)))
	}

	// Extract outgoing metadata from context
	outgoingMD := rk_context.GetOutgoingMD(ctx)
	incomingMD := rk_context.GetIncomingMD(ctx)

	return rk_context.ToContext(ctx, event, incomingMD, outgoingMD, fields)
}

func recordClientAfter(ctx context.Context, opt *Options, err error, method string) rk_query.Event {
	code := opt.errorToCode(err)
	event := rk_context.GetEvent(ctx)
	event.AddErr(err)
	endTime := time.Now()
	elapsed := endTime.Sub(event.GetStartTime())

	if opt.enableLogging(method, err) {
		fields := rk_context.GetFields(ctx)

		// Check whether context is cancelled from server
		select {
		case <-ctx.Done():
			event.AddErr(ctx.Err())
			fields = append(fields, zap.NamedError("server_error", ctx.Err()))
		default:
			break
		}

		// extract request id and log it
		incomingRequestIds := rk_context.GetRequestIdsFromIncomingMD(ctx)
		fields = append(fields,
			zap.String("res_code", code.String()),
			zap.Time("end_time", time.Now()),
			zap.Int64("elapsed_ms", elapsed.Nanoseconds()/1e6),
			zap.Strings("incoming_request_id", incomingRequestIds))

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
		getClientDurationMetrics(method, code.String()).Observe(float64(elapsed.Nanoseconds() / 1e6))
		if err != nil {
			getClientErrorMetrics(method, code.String()).Inc()
		}
		getClientResCodeMetrics(method, code.String()).Inc()
	}

	return event
}
