package rk_context

import (
	"github.com/google/uuid"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"strings"
	"time"
)

const (
	RequestIdKeyLowerCase = "requestid"
	RequestIdKeyDash      = "request-id"
	RequestIdKeyUnderline = "request_id"
	RequestIdKeyDefault   = RequestIdKeyDash
)

type ctxMarker struct{}

type payload struct {
	startTime   time.Time
	queryLogger *zap.Logger
	appLogger   *zap.Logger
	event       rk_query.Event
	incomingMD  *metadata.MD
	outgoingMD  *metadata.MD
	fields      []zapcore.Field
}

var (
	ctxMarkerKey = &ctxMarker{}
	loggerNoop   = zap.NewNop()
)

func IsRkContext(ctx context.Context) bool {
	res := ctx.Value(ctxMarkerKey)
	if res != nil {
		return true
	}

	return false
}

func ToContext(ctx context.Context, queryLogger, appLogger *zap.Logger, event rk_query.Event, incomingMD *metadata.MD, outgoingMD *metadata.MD) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if queryLogger == nil {
		queryLogger = zap.NewNop()
	}

	if appLogger == nil {
		appLogger = zap.NewNop()
	}

	if event == nil {
		event = rk_query.NoopEvent{}
	}

	if incomingMD == nil {
		incomingMD = GetIncomingMD(ctx)
	}

	if outgoingMD == nil {
		outgoingMD = GetOutgoingMD(ctx)
	}

	payload := &payload{
		queryLogger: queryLogger,
		appLogger:   appLogger,
		event:       event,
		incomingMD:  incomingMD,
		outgoingMD:  outgoingMD,
		fields:      extractFields(ctx),
	}

	return context.WithValue(ctx, ctxMarkerKey, payload)
}

// Initialize new context with bellow payloads
//
// 1: zap logger
// 2: event
// 3: incoming metadata
// 4: outgoing metadata
// 5: zap fields
//
// Please do not use it during RPC call or use it with multi thread since it is NOT thread safe
func NewContext(queryLogger *zap.Logger, appLogger *zap.Logger) context.Context {
	base := context.Background()
	incomingMD := GetIncomingMD(base)
	outgoingMD := GetOutgoingMD(base)

	if queryLogger == nil {
		queryLogger = zap.NewNop()
	}

	if appLogger == nil {
		appLogger = zap.NewNop()
	}

	payload := &payload{
		queryLogger: queryLogger,
		appLogger:   appLogger,
		event:       rk_query.NoopEvent{},
		incomingMD:  incomingMD,
		outgoingMD:  outgoingMD,
		fields:      []zap.Field{},
	}

	// Attach incoming and outgoing metadata
	ctx := metadata.NewOutgoingContext(context.Background(), *outgoingMD)
	ctx = metadata.NewIncomingContext(ctx, *incomingMD)

	return context.WithValue(ctx, ctxMarkerKey, payload)
}

// Add Key values to outgoing metadata
//
// We do not recommend to use it as rpc cycle
// It should be used only for common usage
func AddToOutgoingMD(ctx context.Context, key string, values ...string) {
	// for client
	clientMD := GetOutgoingMD(ctx)

	if clientMD != nil {
		clientMD.Append(key, values...)
	}
}

// Add request id to outgoing metadata
//
// The request id would be printed on server's query log and client's query log
// if client is using pulse line gRPC interceptor
func AddRequestIdToOutgoingMD(ctx context.Context) string {
	requestId := GenerateRequestId()

	if len(requestId) > 0 {
		AddToOutgoingMD(ctx, RequestIdKeyDefault, requestId)
	}

	return requestId
}

func GetQueryLogger(ctx context.Context) *zap.Logger {
	payload := getPayload(ctx)

	if payload == nil {
		return loggerNoop
	}

	return payload.queryLogger
}

func GetAppLogger(ctx context.Context) *zap.Logger {
	payload := getPayload(ctx)

	if payload == nil {
		return loggerNoop
	}

	// Add request ids from remote side
	clientRequestIds := GetRequestIdsFromIncomingMD(ctx)
	serverRequestIds := GetRequestIdsFromOutgoingMD(ctx)

	fields := []zap.Field{
		zap.Strings("incoming_request_id", clientRequestIds),
		zap.Strings("outgoing_request_id", serverRequestIds),
	}
	return payload.appLogger.With(fields...)
}

// Extract takes the call-scoped Fields from grpc_zap middleware.
//
// It always returns a Fields that has all the grpc_ctxtags updated.
func GetZapFields(ctx context.Context) []zap.Field {
	payload := getPayload(ctx)

	if payload == nil {
		return []zap.Field{}
	}

	return payload.fields

}

// Extract takes the call-scoped EventData from grpc_zap middleware.
//
// It always returns a EventData that has all the grpc_ctxtags updated.
func GetEvent(ctx context.Context) rk_query.Event {
	payload := getPayload(ctx)

	if payload == nil {
		return rk_query.NoopEvent{}
	}

	return payload.event
}

// Extract takes the call-scoped incoming Metadata from grpc_zap middleware.
//
// It always returns a Metadata that has all the grpc_ctxtags updated.
func GetIncomingMD(ctx context.Context) *metadata.MD {
	payloadRaw := getPayloadRaw(ctx)

	// Payload is empty which means it is not pulse line style context
	// We will try to extract from incoming context
	//
	// If none of them exists, then just return a new empty metadata
	if payloadRaw == nil {
		res, ok := metadata.FromIncomingContext(ctx)
		if ok {
			return &res
		} else {
			md := metadata.Pairs()
			return &md
		}
	}

	payload, ok := payloadRaw.(*payload)

	if !ok || payload == nil {
		md := metadata.Pairs()
		return &md
	}

	return payload.incomingMD
}

// Extract takes the call-scoped outgoing Metadata from grpc_zap middleware.
//
// It always returns a Metadata that has all the grpc_ctxtags updated.
func GetOutgoingMD(ctx context.Context) *metadata.MD {
	payloadRaw := getPayloadRaw(ctx)

	// Payload is empty which means it is not pulse line style context
	// We will try to extract from outging context
	//
	// If none of them exists, then just return a new empty metadata
	if payloadRaw == nil {
		res, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			return &res
		} else {
			md := metadata.Pairs()
			return &md
		}
	}

	payload, ok := payloadRaw.(*payload)

	if !ok || payload == nil {
		md := metadata.Pairs()
		return &md
	}

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

// Extract takes the call-scoped outging Metadata from grpc_zap middleware.
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

// Retrieve context if possible
func getPayloadRaw(ctx context.Context) interface{} {
	if ctx == nil {
		return nil
	}

	return ctx.Value(ctxMarkerKey)
}

// Retrieve pulse line context payload if possible
func getPayload(ctx context.Context) *payload {
	if ctx == nil {
		return nil
	}

	payload, ok := ctx.Value(ctxMarkerKey).(*payload)

	if !ok || payload == nil {
		return nil
	}

	return payload
}

func extractFields(ctx context.Context) []zap.Field {
	payloadRaw := getPayloadRaw(ctx)

	// Payload is empty which means it is not pulse line style context
	// We will try to extract from incoming context
	//
	// If none of them exists, then just return a new empty metadata
	if payloadRaw == nil {
		return []zap.Field{}
	}

	payload, ok := payloadRaw.(*payload)

	if !ok || payload == nil {
		return []zap.Field{}
	}

	return payload.fields
}
