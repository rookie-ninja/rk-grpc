// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcctx

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-logger"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"net"
	"strings"
)

var (
	Realm         = zap.String("realm", rkcommon.GetEnvValueOrDefault("REALM", "unknown"))
	Region        = zap.String("region", rkcommon.GetEnvValueOrDefault("REGION", "unknown"))
	AZ            = zap.String("az", rkcommon.GetEnvValueOrDefault("AZ", "unknown"))
	Domain        = zap.String("domain", rkcommon.GetEnvValueOrDefault("DOMAIN", "unknown"))
	LocalIp       = zap.String("localIp", rkcommon.GetLocalIP())
	LocalHostname = zap.String("localHostname", rkcommon.GetLocalHostname())
)

const (
	RequestIdKeyLowerCase = "requestid"
	RequestIdKeyDash      = "request-id"
	RequestIdKeyUnderline = "request_id"
	RequestIdKeyDefault   = RequestIdKeyDash
	RkEventKey            = "rkEvent"
	RkLoggerKey           = "rkLogger"
	RkEntryNameKey        = "rkEntryKey"
	RkEntryNameValue      = "rkEntry"
	RkEntryTypeValue      = "grpc"
	RpcTypeUnaryServer    = "unaryServer"
	RpcTypeStreamServer   = "streamServer"
	RpcTypeUnaryClient    = "unaryClient"
	RpcTypeStreamClient   = "streamClient"
)

type RpcInfo struct {
	Target      string
	GrpcService string
	GrpcMethod  string
	GwPath      string
	GwMethod    string
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
	zapLogger  *zap.Logger
	incomingMD metadata.MD
	outgoingMD metadata.MD
}

var (
	key = &payloadKey{}
)

type OptionSet interface {
	GetEntryName() string

	GetEntryType() string
}

func ErrorToCodesFuncDefault(err error) codes.Code {
	return status.Code(err)
}

type FakeGRPCEntry struct{}

func NewFakeGRPCEntry() *FakeGRPCEntry {
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

func ContainsPayload(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	res := ctx.Value(key)
	if res != nil {
		return true
	}

	return false
}

type ContextOption func(*payload)

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
			load.zapLogger = logger
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

func ContextWithPayload(ctx context.Context, opts ...ContextOption) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	var load *payload

	if ContainsPayload(ctx) {
		load = GetPayload(ctx)
	} else {
		load = &payload{
			event:      rkentry.NoopEventLoggerEntry().GetEventFactory().CreateEventNoop(),
			zapLogger:  rklogger.NoopLogger,
			entryName:  NewFakeGRPCEntry().GetName(),
			incomingMD: metadata.Pairs(),
			outgoingMD: metadata.Pairs(),
		}
	}

	for i := range opts {
		opts[i](load)
	}

	return context.WithValue(ctx, key, load)
}

// Initialize new context with bellow payloads
// Please do not use it during RPC call or use it with multi thread since it is NOT thread safe
func NewContext() context.Context {
	base := context.Background()
	incomingMD := GetIncomingMD(base)
	outgoingMD := GetOutgoingMD(base)

	payload := &payload{
		event:      rkquery.NewEventFactory().CreateEventNoop(),
		zapLogger:  rklogger.NoopLogger,
		entryName:  NewFakeGRPCEntry().GetName(),
		incomingMD: incomingMD,
		outgoingMD: outgoingMD,
	}

	// Attach incoming and outgoing metadata
	ctx := metadata.NewOutgoingContext(context.Background(), outgoingMD)
	ctx = metadata.NewIncomingContext(ctx, incomingMD)

	return context.WithValue(ctx, key, payload)
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

// Add request id to outgoing metadata
//
// The request id would be printed on server's query log and client's query log
// if client is using rk gRPC interceptor
func AddRequestIdToOutgoingMD(ctx context.Context) string {
	requestId := GenerateRequestId()

	if len(requestId) > 0 {
		AddToOutgoingMD(ctx, RequestIdKeyDefault, requestId)
		payload := GetPayload(ctx)
		payload.zapLogger = payload.zapLogger.With(zap.Strings("outgoingRequestId", GetValueFromOutgoingMD(ctx, RequestIdKeyDefault)))
	}

	return requestId
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
func GetZapLogger(ctx context.Context) *zap.Logger {
	if !IsRkContext(ctx) {
		return rklogger.NoopLogger
	}

	payload := GetPayload(ctx)
	return payload.zapLogger
}

func SetZapLogger(ctx context.Context, logger *zap.Logger) {
	if !IsRkContext(ctx) {
		return
	}
	payload := GetPayload(ctx)
	payload.zapLogger = logger
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

func GetRequestIdsFromOutgoingMD(ctx context.Context) []string {
	dash := GetValueFromOutgoingMD(ctx, RequestIdKeyDash)
	underLine := GetValueFromOutgoingMD(ctx, RequestIdKeyUnderline)
	lower := GetValueFromOutgoingMD(ctx, RequestIdKeyLowerCase)

	res := make([]string, 0)

	res = append(res, dash...)
	res = append(res, underLine...)
	res = append(res, lower...)

	return res
}

func GetRequestIdsFromIncomingMD(ctx context.Context) []string {
	dash := GetValueFromIncomingMD(ctx, RequestIdKeyDash)
	underLine := GetValueFromIncomingMD(ctx, RequestIdKeyUnderline)
	lower := GetValueFromIncomingMD(ctx, RequestIdKeyLowerCase)

	res := make([]string, 0)

	res = append(res, dash...)
	res = append(res, underLine...)
	res = append(res, lower...)

	return res
}

// Generate request id based on google/uuid
// UUIDs are based on RFC 4122 and DCE 1.1: Authentication and Security Services.
//
// A UUID is a 16 byte (128 bit) array. UUIDs may be used as keys to maps or compared directly.
func GenerateRequestId() string {
	// Do not use uuid.New() since it would panic if any error occurs
	requestId, err := uuid.NewRandom()

	// Currently, we will return empty string if error occurs
	if err != nil {
		return ""
	}

	return requestId.String()
}

// Generate request id based on google/uuid
// UUIDs are based on RFC 4122 and DCE 1.1: Authentication and Security Services.
//
// A UUID is a 16 byte (128 bit) array. UUIDs may be used as keys to maps or compared directly.
func GenerateRequestIdWithPrefix(prefix string) string {
	// Do not use uuid.New() since it would panic if any error occurs
	requestId, err := uuid.NewRandom()

	// Currently, we will return empty string if error occurs
	if err != nil {
		return ""
	}

	return prefix + "-" + requestId.String()
}

// Generate request id with prefix.
func GenerateTraceIdWithPrefix(prefix string) string {
	return GenerateRequestIdWithPrefix(prefix)
}

// Retrieve context if possible.
func getPayloadRaw(ctx context.Context) interface{} {
	if ctx == nil {
		return nil
	}

	return ctx.Value(key)
}

// Retrieve rk context payload if possible.
func GetPayload(ctx context.Context) *payload {
	if ctx == nil {
		return &payload{
			event:      rkquery.NewEventFactory().CreateEventNoop(),
			entryName:  NewFakeGRPCEntry().GetName(),
			zapLogger:  rklogger.NoopLogger,
			incomingMD: GetIncomingMD(ctx),
			outgoingMD: GetOutgoingMD(ctx),
		}
	}

	val, ok := ctx.Value(key).(*payload)

	if !ok || val == nil {
		return &payload{
			event:      rkquery.NewEventFactory().CreateEventNoop(),
			entryName:  NewFakeGRPCEntry().GetName(),
			zapLogger:  rklogger.NoopLogger,
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
		Target:      "unknown",
	}
}

// Get remote endpoint information set including IP, Port, NetworkType
// We will do as best as we can to determine it
// If fails, then just return default ones
func GetRemoteAddressSetAsFields(ctx context.Context) []zap.Field {
	ip, port, netType := GetRemoteAddressSet(ctx)

	return []zap.Field{
		zap.String("remoteIp", ip),
		zap.String("remotePort", port),
		zap.String("remoteNetType", netType),
	}
}

func GetRemoteAddressSet(ctx context.Context) (ip, port, netType string) {
	remoteIp := "0.0.0.0"
	remotePort := "0"
	remoteNetworkType := ""
	if peer, ok := peer.FromContext(ctx); ok {
		remoteNetworkType = peer.Addr.Network()

		// Here is the tricky part
		// We only try to parse IPV4 style Address
		// Rest of peer.Addr implementations are not well formatted string
		// and in this case, we leave port as zero and IP as the returned
		// String from Addr.String() function
		//
		// BTW, just skip the error since it would not impact anything
		// Operators could observe this error from monitor dashboards by
		// validating existence of IP & PORT fields
		remoteIp, remotePort, _ = net.SplitHostPort(peer.Addr.String())
	}

	headers, ok := metadata.FromIncomingContext(ctx)

	if ok {
		forwardedRemoteIPList := headers["x-forwarded-for"]

		// Deal with forwarded remote ip
		if len(forwardedRemoteIPList) > 0 {
			forwardedRemoteIP := forwardedRemoteIPList[0]

			if forwardedRemoteIP == "::1" {
				forwardedRemoteIP = "localhost"
			}

			remoteIp = forwardedRemoteIP
		}
	}

	if remoteIp == "::1" {
		remoteIp = "localhost"
	}

	return remoteIp, remotePort, remoteNetworkType
}

func ToOptionsKey(entryName, rpcType string) string {
	return strings.Join([]string{entryName, rpcType}, "-")
}
