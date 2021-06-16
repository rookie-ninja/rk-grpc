// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rkgrpcbasic

import (
	"context"
	"github.com/rookie-ninja/rk-common/common"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"net"
	"strings"
)

const (
	RkEntryNameKey      = "rkEntryKey"
	RkEntryNameValue    = "rkEntry"
	RkEntryTypeValue    = "grpc"
	RpcTypeUnaryServer  = "unaryServer"
	RpcTypeStreamServer = "streamServer"
	RpcTypeUnaryClient  = "unaryClient"
	RpcTypeStreamClient = "streamClient"
)

var (
	Realm         = zap.String("realm", rkcommon.GetEnvValueOrDefault("REALM", "*"))
	Region        = zap.String("region", rkcommon.GetEnvValueOrDefault("REGION", "*"))
	AZ            = zap.String("az", rkcommon.GetEnvValueOrDefault("AZ", "*"))
	Domain        = zap.String("domain", rkcommon.GetEnvValueOrDefault("DOMAIN", "*"))
	LocalIp       = zap.String("localIp", rkcommon.GetLocalIP())
	LocalHostname = zap.String("localHostname", rkcommon.GetLocalHostname())
)

// Interceptor would distinguish metrics set based on.
var optionsMap = make(map[string]*optionSet)

func newOptionSet(rpcType string, opts ...Option) *optionSet {
	set := &optionSet{
		EntryName: RkEntryNameValue,
		EntryType: RkEntryTypeValue,
	}

	for i := range opts {
		opts[i](set)
	}

	key := ToOptionsKey(set.EntryName, rpcType)
	if _, ok := optionsMap[key]; !ok {
		optionsMap[key] = set
	}

	return set
}

// options which is used while initializing basic interceptor
type optionSet struct {
	EntryName string
	EntryType string
}

type Option func(*optionSet)

func WithEntryNameAndType(entryName, entryType string) Option {
	return func(set *optionSet) {
		if len(entryName) > 0 {
			set.EntryName = entryName
		}
		if len(entryType) > 0 {
			set.EntryType = entryType
		}
	}
}

func ToOptionsKey(entryName, rpcType string) string {
	return strings.Join([]string{entryName, rpcType}, "-")
}

// Read remote Ip and port from metadata.
// If user enabled RK style gateway server mux option, then there would be bellow headers forwarded
// to grpc metadata
// 1: x-forwarded-method
// 2: x-forwarded-path
// 3: x-forwarded-scheme
// 4: x-forwarded-user-agent
// 5: x-forwarded-remote-addr
func getRemoteAddressSetFromMeta(md metadata.MD) (ip, port string) {
	if v := md.Get("x-forwarded-remote-addr"); len(v) > 0 {
		ip, port, _ = net.SplitHostPort(v[0])
	}

	if ip == "::1" {
		ip = "localhost"
	}

	return ip, port
}

// Read remote Ip and port from metadata first.
func getRemoteAddressSet(ctx context.Context) (ip, port, netType string) {
	md, _ := metadata.FromIncomingContext(ctx)
	ip, port = getRemoteAddressSetFromMeta(md)
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
