// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpcinter provides common utility functions for middleware of grpc framework
package rkgrpcinter

import (
	"context"
	"github.com/rookie-ninja/rk-common/common"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"net"
	"path"
	"strings"
)

var (
	// Realm environment variable
	Realm = zap.String("realm", rkcommon.GetEnvValueOrDefault("REALM", "*"))
	// Region environment variable
	Region = zap.String("region", rkcommon.GetEnvValueOrDefault("REGION", "*"))
	// AZ environment variable
	AZ = zap.String("az", rkcommon.GetEnvValueOrDefault("AZ", "*"))
	// Domain environment variable
	Domain = zap.String("domain", rkcommon.GetEnvValueOrDefault("DOMAIN", "*"))
	// LocalIp read local IP from localhost
	LocalIp = zap.String("localIp", rkcommon.GetLocalIP())
	// LocalHostname read hostname from localhost
	LocalHostname = zap.String("localHostname", rkcommon.GetLocalHostname())

	clientPayloadKey = &clientPayload{}
	serverPayloadKey = &serverPayload{}
)

const (
	// RpcEntryNameKey default entry name key of grpc
	RpcEntryNameKey = "grpcEntryName"
	// RpcEntryNameValue default entry name value of grpc
	RpcEntryNameValue = "grpc"
	// RpcEntryTypeKey default entry type key of grpc
	RpcEntryTypeKey = "grpcEntryType"
	// RpcEntryTypeValue default entry type value of grpc
	RpcEntryTypeValue = "grpc"

	// RpcEventKey event key in grpc context
	RpcEventKey = "grpcEvent"
	// RpcLoggerKey logger key in grpc context
	RpcLoggerKey = "grpcLogger"
	// RpcMethodKey RPC method key in grpc context
	RpcMethodKey = "grpcMethod"
	// RpcErrorKey RPC error key in grpc context
	RpcErrorKey = "grpcError"

	// RpcTypeKey RPC type key in grpc context
	RpcTypeKey = "grpcType"
	// RpcTracerKey tracer key in grpc context
	RpcTracerKey = "grpcTracer"
	// RpcSpanKey Span key in grpc context
	RpcSpanKey = "grpcSpan"
	// RpcTracerProviderKey tracer provider key in grpc context
	RpcTracerProviderKey = "grpcTracerProvider"
	// RpcPropagatorKey propagator key in grpc context
	RpcPropagatorKey = "grpcPropagator"

	// RpcAuthorizationHeaderKey basic auth header key in request
	RpcAuthorizationHeaderKey = "authorization"
	// RpcApiKeyHeaderKey api key header in request
	RpcApiKeyHeaderKey = "X-API-Key"

	// RpcTypeUnaryServer unary RPC server type key in context
	RpcTypeUnaryServer = "unaryServer"
	// RpcTypeStreamServer stream RPC server type key in context
	RpcTypeStreamServer = "streamServer"
	// RpcTypeUnaryClient unary RPC client type key in context
	RpcTypeUnaryClient = "unaryClient"
	// RpcTypeStreamClient stream RPC client type in context
	RpcTypeStreamClient = "streamClient"
	// RpcJwtTokenKey key of jwt token in context
	RpcJwtTokenKey = "grpcJwt"
	// RpcCsrfTokenKey key of csrf token injected by csrf middleware
	RpcCsrfTokenKey = "grpcCsrfToken"
)

// RpcPayloadAppended a flag used in inner middleware
var RpcPayloadAppended = rpcPayloadAppended{}

type rpcPayloadAppended struct{}

type clientPayload struct {
	incomingHeaders *metadata.MD
	outgoingHeaders *metadata.MD
	m               map[string]interface{}
}

type serverPayload struct {
	m map[string]interface{}
}

// GetGwInfo Extract gateway related information from metadata.
func GetGwInfo(md metadata.MD) (gwMethod, gwPath, gwScheme, gwUserAgent string) {
	gwMethod, gwPath, gwScheme, gwUserAgent = "", "", "", ""

	if tokens := md["x-forwarded-method"]; len(tokens) > 0 {
		gwMethod = tokens[0]
	}

	if tokens := md["x-forwarded-path"]; len(tokens) > 0 {
		gwPath = tokens[0]
	}

	if tokens := md["x-forwarded-scheme"]; len(tokens) > 0 {
		gwScheme = tokens[0]
	}

	if tokens := md["x-forwarded-user-agent"]; len(tokens) > 0 {
		gwUserAgent = tokens[0]
	}

	return gwMethod, gwPath, gwScheme, gwUserAgent
}

// GetGrpcInfo Extract grpc related information from fullMethod.
func GetGrpcInfo(fullMethod string) (grpcService, grpcMethod string) {
	// Parse rpc path info including gateway
	grpcService, grpcMethod = "", ""

	if v := strings.TrimPrefix(path.Dir(fullMethod), "/"); len(v) > 0 {
		grpcService = v
	}

	if v := strings.TrimPrefix(path.Base(fullMethod), "/"); len(v) > 0 {
		grpcMethod = v
	}

	return grpcService, grpcMethod
}

// ToOptionsKey Convert to optionsMap key with entry name and rpcType.
func ToOptionsKey(entryName, rpcType string) string {
	return strings.Join([]string{entryName, rpcType}, "-")
}

// GetRemoteAddressSetFromMeta Read remote Ip and port from metadata.
// If user enabled RK style gateway server mux option, then there would be bellow headers forwarded
// to grpc metadata
// 1: x-forwarded-method
// 2: x-forwarded-path
// 3: x-forwarded-scheme
// 4: x-forwarded-user-agent
// 5: x-forwarded-remote-addr
func GetRemoteAddressSetFromMeta(md metadata.MD) (ip, port string) {
	if v := md.Get("x-forwarded-remote-addr"); len(v) > 0 {
		ip, port, _ = net.SplitHostPort(v[0])
	}

	if ip == "::1" {
		ip = "localhost"
	}

	return ip, port
}

// GetRemoteAddressSet Read remote Ip and port from metadata first.
func GetRemoteAddressSet(ctx context.Context) (ip, port, netType string) {
	md, _ := metadata.FromIncomingContext(ctx)
	ip, port = GetRemoteAddressSetFromMeta(md)
	// no ip and port were passed through gateway

	if len(ip) < 1 {
		ip, port, netType = "0.0.0.0", "0", ""
		if peer, ok := peer.FromContext(ctx); ok {
			netType = peer.Addr.Network()
			// Here is the tricky part
			// We only try to parse IPV4 style Address
			// Rest of peer.Addr implementations are not well formatted string
			// and in this case, we leave port as zero and IP as the returned
			// String from Addr.String() function
			//
			// BTW, just skip the error since it would not impact anything
			// Operators could observe this error from monitor dashboards by
			// validating existence of IP & PORT fields
			ip, port, _ = net.SplitHostPort(peer.Addr.String())
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

				ip = forwardedRemoteIP
			}
		}

		if ip == "::1" {
			ip = "localhost"
		}
	}

	return ip, port, netType
}

// MergeToOutgoingMD Merge md to context outgoing metadata.
func MergeToOutgoingMD(ctx context.Context, md metadata.MD) context.Context {
	if appended := ctx.Value(RpcPayloadAppended); appended == nil {
		if _, ok := metadata.FromOutgoingContext(ctx); ok {
			kvs := make([]string, 0)
			// append md into result first
			for k, v := range md {
				for i := range v {
					kvs = append(kvs, k, v[i])
				}
			}
			ctx = context.WithValue(ctx, RpcPayloadAppended, "")

			// merge incoming MD into outgoing metadata
			ctx = metadata.AppendToOutgoingContext(ctx, kvs...)
		} else {
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
	}

	return ctx
}

// MergeAndDeduplicateSlice Merge src and targets and deduplicate
func MergeAndDeduplicateSlice(src []string, target []string) []string {
	m := make(map[string]bool)
	for i := range src {
		m[src[i]] = true
	}

	for i := range target {
		if _, ok := m[target[i]]; !ok {
			src = append(src, target[i])
		}
	}

	return src
}

// ***************************************************
// ********** Internal usage for context *************
// ***************************************************

// WrapContextForServer Wrap server context.
func WrapContextForServer(ctx context.Context) context.Context {
	if v := ctx.Value(serverPayloadKey); v != nil {
		return ctx
	}

	return context.WithValue(ctx, serverPayloadKey, serverPayload{
		m: make(map[string]interface{}),
	})
}

// GetIncomingHeadersOfClient get incoming header from client
func GetIncomingHeadersOfClient(ctx context.Context) *metadata.MD {
	if v := ctx.Value(clientPayloadKey); v != nil {
		// called from client side
		return v.(clientPayload).incomingHeaders
	}

	md := metadata.Pairs()
	return &md
}

// GetOutgoingHeadersOfClient get outgoing header from client
func GetOutgoingHeadersOfClient(ctx context.Context) *metadata.MD {
	if v := ctx.Value(clientPayloadKey); v != nil {
		// called from client side
		return v.(clientPayload).outgoingHeaders
	}

	md := metadata.Pairs()
	return &md
}

// GetServerContextPayload get context payload injected into server side context
func GetServerContextPayload(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return make(map[string]interface{})
	}

	if v := ctx.Value(serverPayloadKey); v != nil {
		return v.(serverPayload).m
	}

	return make(map[string]interface{})
}

// AddToServerContextPayload add k/v into payload injected into server side context
func AddToServerContextPayload(ctx context.Context, key string, value interface{}) {
	if value != nil {
		GetServerContextPayload(ctx)[key] = value
	}
}

// GetClientContextPayload get payload injected into client side context
func GetClientContextPayload(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return make(map[string]interface{})
	}

	if v := ctx.Value(clientPayloadKey); v != nil {
		return v.(clientPayload).m
	}

	return make(map[string]interface{})
}

// AddToClientContextPayload add k/v into payload injected into client side context
func AddToClientContextPayload(ctx context.Context, key string, value interface{}) {
	if value != nil {
		GetClientContextPayload(ctx)[key] = value
	}
}

// ContainsClientPayload is payload injected into client side context?
func ContainsClientPayload(ctx context.Context) bool {
	if v := ctx.Value(clientPayloadKey); v != nil {
		return true
	}

	return false
}

// ContainsServerPayload is payload injected into server side context?
func ContainsServerPayload(ctx context.Context) bool {
	if v := ctx.Value(serverPayloadKey); v != nil {
		return true
	}

	return false
}

// NewClientPayload create new client side payload
func NewClientPayload() clientPayload {
	incomingMD := metadata.Pairs()
	outgoingMD := metadata.Pairs()

	return clientPayload{
		incomingHeaders: &incomingMD,
		outgoingHeaders: &outgoingMD,
		m:               make(map[string]interface{}),
	}
}

// GetClientPayloadKey get client payload key used in context.Context
func GetClientPayloadKey() interface{} {
	return clientPayloadKey
}

// GetServerPayloadKey get server payload key used in context.Context
func GetServerPayloadKey() interface{} {
	return serverPayloadKey
}
