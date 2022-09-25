package rkgrpc

import (
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"regexp"
	"strings"
	"time"
)

type BootConfigGrpcWeb struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Cors    struct {
		AllowOrigins []string `yaml:"allowOrigins" json:"allowOrigins"`
	} `yaml:"cors" json:"cors"`
	Websocket struct {
		Enabled               bool   `yaml:"enabled" json:"enabled"`
		PingIntervalMs        int64  `yaml:"pingIntervalMs" json:"pingIntervalMs"`
		MessageReadLimitBytes int64  `yaml:"messageReadLimitBytes" json:"messageReadLimitBytes"`
		CompressMode          string `yaml:"compressMode" json:"compressMode"`
	} `yaml:"websocket" json:"websocket"`
}

func ToAllowOriginFunc(allowOrigins []string) grpcweb.Option {
	if len(allowOrigins) < 1 {
		return grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		})
	}

	allowPattern := make([]string, 0)
	for i := range allowOrigins {
		var result strings.Builder
		result.WriteString("^")
		for j, literal := range strings.Split(allowOrigins[i], "*") {

			// Replace * with .*
			if j > 0 {
				result.WriteString(".*")
			}

			result.WriteString(literal)
		}
		result.WriteString("$")
		allowPattern = append(allowPattern, result.String())
	}

	return grpcweb.WithOriginFunc(func(origin string) bool {
		for _, pattern := range allowPattern {
			res, _ := regexp.MatchString(pattern, origin)
			if res {
				break
			}
		}
		return false
	})
}

func ToGrpcWebOptions(conf *BootConfigGrpcWeb) []grpcweb.Option {
	opt := make([]grpcweb.Option, 0)
	opt = append(opt, ToAllowOriginFunc(conf.Cors.AllowOrigins))

	if conf.Websocket.Enabled {
		opt = append(opt, grpcweb.WithWebsockets(true))
		if conf.Websocket.PingIntervalMs > 0 {
			opt = append(opt, grpcweb.WithWebsocketPingInterval(time.Duration(conf.Websocket.PingIntervalMs)*time.Microsecond))
		}
		if conf.Websocket.MessageReadLimitBytes > 0 {
			opt = append(opt, grpcweb.WithWebsocketsMessageReadLimit(conf.Websocket.MessageReadLimitBytes))
		}
	}

	return opt
}
