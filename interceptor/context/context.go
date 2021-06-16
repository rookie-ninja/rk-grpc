// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcctx

import (
	"encoding/json"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"strings"
)

var (
	key                  = &payloadKey{}
	RequestIdMetadataKey = "X-RK-Request-Id"
)

type RpcInfo struct {
	GrpcService string
	GrpcMethod  string
	GwPath      string
	GwMethod    string
	GwScheme    string
	GwUserAgent string
	Type        string
	RemoteIp    string
	RemotePort  string
	Err         error
}

type payloadKey struct{}

type payload struct {
	event      rkquery.Event
	entryName  string
	rpcInfo    *RpcInfo
	logger     *zap.Logger
	tracer     trace.Tracer
	provider   trace.TracerProvider
	propagator propagation.TextMapPropagator
	incomingMD metadata.MD
	outgoingMD metadata.MD
}

type FakeGRPCEntry struct{}

type ContextOption func(*payload)

func WithTracer(in trace.Tracer) ContextOption {
	return func(load *payload) {
		load.tracer = in
	}
}

func WithTraceProvider(in *sdktrace.TracerProvider) ContextOption {
	return func(load *payload) {
		load.provider = in
	}
}

func WithPropagator(in propagation.TextMapPropagator) ContextOption {
	return func(load *payload) {
		load.propagator = in
	}
}

func WithEvent(event rkquery.Event) ContextOption {
	return func(load *payload) {
		if event != nil {
			load.event = event
		}
	}
}

func WithZapLogger(logger *zap.Logger) ContextOption {
	return func(load *payload) {
		if logger != nil {
			load.logger = logger
		}
	}
}

func WithEntryName(entryName string) ContextOption {
	return func(load *payload) {
		load.entryName = entryName
	}
}

func WithIncomingMD(incomingMD metadata.MD) ContextOption {
	return func(load *payload) {
		if incomingMD != nil {
			load.incomingMD = incomingMD
		}
	}
}

func WithOutgoingMD(outgoingMD metadata.MD) ContextOption {
	return func(load *payload) {
		if outgoingMD != nil {
			load.outgoingMD = outgoingMD
		}
	}
}

func WithRpcInfo(info *RpcInfo) ContextOption {
	return func(load *payload) {
		if info != nil {
			load.rpcInfo = info
		}
	}
}

// Initialize new context with bellow payloads
// Please do not use it during RPC call or use it with multi thread since it is NOT thread safe
func NewContext() context.Context {
	base := context.Background()
	incomingMD := GetIncomingMD(base)
	outgoingMD := GetOutgoingMD(base)

	payload := &payload{
		event:      rkquery.NewEventFactory().CreateEventNoop(),
		logger:     rklogger.NoopLogger,
		entryName:  newFakeGRPCEntry().GetName(),
		incomingMD: incomingMD,
		outgoingMD: outgoingMD,
	}

	// Attach incoming and outgoing metadata
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)
	ctx = metadata.NewIncomingContext(ctx, incomingMD)

	return context.WithValue(ctx, key, payload)
}

func ToRkContext(ctx context.Context, opts ...ContextOption) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	var load *payload

	if containsPayload(ctx) {
		load = GetPayload(ctx)
	} else {
		load = &payload{
			event:      rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEventNoop(),
			logger:     rklogger.NoopLogger,
			entryName:  newFakeGRPCEntry().GetName(),
			incomingMD: metadata.Pairs(),
			outgoingMD: metadata.Pairs(),
		}
	}

	for i := range opts {
		opts[i](load)
	}

	return context.WithValue(ctx, key, load)
}

func IsRkContext(ctx context.Context) bool {
	if ctx == nil || getPayloadRaw(ctx) == nil {
		return false
	}

	return true
}

// Add Key values to outgoing metadata
//
// We do not recommend to use it as rpc cycle
// It should be used only for common usage
func AddToOutgoingMD(ctx context.Context, key string, values ...string) {
	if ctx == nil || !IsRkContext(ctx) {
		return
	}

	// for client
	clientMD := GetOutgoingMD(ctx)

	// Is it ok to append values to outgoing metadata?
	// Grpc suggest as bellow:
	// The returned MD should not be modified.
	//
	// However, since we store outgoing metadata in payload, so it is safe to do as bellow.
	if clientMD != nil {
		clientMD.Append(key, values...)
	}
}

// Set Key values to outgoing metadata
//
// We do not recommend to use it as rpc cycle
// It should be used only for common usage
func SetToOutgoingMD(ctx context.Context, key string, values ...string) {
	if ctx == nil || !IsRkContext(ctx) {
		return
	}

	// for client
	clientMD := GetOutgoingMD(ctx)

	// Is it ok to append values to outgoing metadata?
	// Grpc suggest as bellow:
	// The returned MD should not be modified.
	//
	// However, since we store outgoing metadata in payload, so it is safe to do as bellow.
	if clientMD != nil {
		clientMD.Set(key, values...)
	}
}

// Add request id to outgoing metadata
//
// The request id would be printed on server's query log and client's query log
// if client is using rk gRPC interceptor
func SetRequestIdToOutgoingMD(ctx context.Context) string {
	requestId := rkcommon.GenerateRequestId()

	if len(requestId) > 0 {
		// Do not call AddToOutgoingMD(), we need to make sure there is only one requestId globally.
		SetToOutgoingMD(ctx, RequestIdMetadataKey, requestId)
		event := GetEvent(ctx)
		event.SetRequestId(requestId)
		event.SetEventId(requestId)
	}

	return requestId
}

// Extract takes the call-scoped Tracer from middleware.
func GetTracerProvider(ctx context.Context) trace.TracerProvider {
	if !IsRkContext(ctx) {
		return trace.NewNoopTracerProvider()
	}

	payload := GetPayload(ctx)
	return payload.provider
}

// Extract takes the call-scoped span processor from middleware.
func GetTracerPropagator(ctx context.Context) propagation.TextMapPropagator {
	if !IsRkContext(ctx) {
		return nil
	}

	payload := GetPayload(ctx)
	return payload.propagator
}

// Extract takes the call-scoped EventData from grpc_zap middleware.
// Returns noop event if context is nil or is not RK style.
//
// It always returns a EventData that has all the grpc_ctxtags updated.
func GetEvent(ctx context.Context) rkquery.Event {
	if !IsRkContext(ctx) {
		return rkquery.NewEventFactory().CreateEventNoop()
	}

	payload := GetPayload(ctx)
	return payload.event
}

// Extract takes the call-scoped zap logger from grpc_zap middleware.
//
// It always returns a zap logger that has all the grpc_ctxtags updated.
func GetLogger(ctx context.Context) *zap.Logger {
	if !IsRkContext(ctx) {
		return rklogger.NoopLogger
	}

	payload := GetPayload(ctx)

	return payload.logger.With(
		zap.String("traceId", GetTraceId(ctx)), zap.String("requestId", GetRequestId(ctx)))
}

func SetLogger(ctx context.Context, logger *zap.Logger) {
	if !IsRkContext(ctx) {
		return
	}
	payload := GetPayload(ctx)
	payload.logger = logger
}

func GetRequestId(ctx context.Context) string {
	if v := GetValueFromOutgoingMD(ctx, RequestIdMetadataKey); len(v) > 0 {
		return v[0]
	}
	return ""
}

func GetTraceId(ctx context.Context) string {
	if v := ctx.Value("rk-trace-id"); v != nil {
		return v.(string)
	}

	return ""
}

// Extract takes the call-scoped incoming Metadata from grpc_zap middleware.
//
// It always returns a Metadata that has all the grpc_ctxtags updated.
func GetIncomingMD(ctx context.Context) metadata.MD {
	if !IsRkContext(ctx) {
		if ctx == nil {
			return metadata.Pairs()
		}

		// It is not rk style context
		// We will try to extract from incoming context
		//
		// If none of them exists, then just return a new empty metadata
		res, ok := metadata.FromIncomingContext(ctx)
		if ok {
			return res
		} else {
			md := metadata.Pairs()
			return md
		}
	}

	payloadRaw := getPayloadRaw(ctx)
	payload := payloadRaw.(*payload)

	return payload.incomingMD
}

// Extract takes the call-scoped outgoing Metadata from grpc_zap middleware.
//
// It always returns a Metadata that has all the grpc_ctxtags updated.
// Please do not modify metadata if context is not RK style.
//
// If context is RK style, then use AddToOutgoingMD instead.
func GetOutgoingMD(ctx context.Context) metadata.MD {
	if !IsRkContext(ctx) {
		if ctx == nil {
			return metadata.Pairs()
		}

		// It is not rk style context
		// We will try to extract from incoming context
		//
		// If none of them exists, then just return a new empty metadata
		res, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			return res
		} else {
			md := metadata.Pairs()
			return md
		}
	}

	payloadRaw := getPayloadRaw(ctx)
	payload := payloadRaw.(*payload)

	return payload.outgoingMD
}

// Extract takes the call-scoped incoming Metadata from grpc_zap middleware.
//
// It always returns a Metadata that has all the grpc_ctxtags updated.
func GetValueFromIncomingMD(ctx context.Context, key string) []string {
	md := GetIncomingMD(ctx)

	if md == nil {
		return []string{}
	}

	values := md.Get(strings.ToLower(key))

	return values
}

// Extract takes the call-scoped outgoing Metadata from grpc_zap middleware.
//
// It always returns a Metadata that has all the grpc_ctxtags updated.
func GetValueFromOutgoingMD(ctx context.Context, key string) []string {
	md := GetOutgoingMD(ctx)

	if md == nil {
		return []string{}
	}

	value := md.Get(strings.ToLower(key))

	return value
}

// Retrieve rk context payload if possible.
func GetPayload(ctx context.Context) *payload {
	if ctx == nil {
		return &payload{
			event:      rkquery.NewEventFactory().CreateEventNoop(),
			entryName:  newFakeGRPCEntry().GetName(),
			logger:     rklogger.NoopLogger,
			incomingMD: GetIncomingMD(ctx),
			outgoingMD: GetOutgoingMD(ctx),
		}
	}

	val, ok := ctx.Value(key).(*payload)

	if !ok || val == nil {
		return &payload{
			event:      rkquery.NewEventFactory().CreateEventNoop(),
			entryName:  newFakeGRPCEntry().GetName(),
			logger:     rklogger.NoopLogger,
			incomingMD: GetIncomingMD(ctx),
			outgoingMD: GetOutgoingMD(ctx),
		}
	}

	return val
}

// Retrieve entry name in payload if possible.
func GetEntryName(ctx context.Context) string {
	if load := GetPayload(ctx); load != nil {
		return load.entryName
	}

	return ""
}

// Retrieve rpcInfo if possible.
func GetRpcInfo(ctx context.Context) *RpcInfo {
	payload := GetPayload(ctx)

	if payload != nil {
		return payload.rpcInfo
	}

	return &RpcInfo{
		Type:        "unknown",
		GrpcService: "unknown",
		GrpcMethod:  "unknown",
		GwMethod:    "unknown",
		GwPath:      "unknown",
		GwScheme:    "unknown",
		GwUserAgent: "unknown",
	}
}

// Retrieve context if possible.
func getPayloadRaw(ctx context.Context) interface{} {
	if ctx == nil {
		return nil
	}

	return ctx.Value(key)
}

func containsPayload(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	res := ctx.Value(key)
	if res != nil {
		return true
	}

	return false
}

func newFakeGRPCEntry() *FakeGRPCEntry {
	return &FakeGRPCEntry{}
}

func (entry *FakeGRPCEntry) Bootstrap(context.Context) {}

func (entry *FakeGRPCEntry) Interrupt(context.Context) {}

func (entry *FakeGRPCEntry) String() string {
	m := map[string]interface{}{
		"entryName": "fakeGrpcEntryName",
		"entryType": "fakeGrpcEntry",
	}

	bytes, _ := json.Marshal(m)

	return string(bytes)
}

func (entry *FakeGRPCEntry) GetName() string {
	return "fakeGrpcEntryName"
}

func (entry *FakeGRPCEntry) GetType() string {
	return "fakeGrpcEntry"
}

func (entry *FakeGRPCEntry) GetDescription() string {
	return "fakeGrpcEntryDescription"
}
