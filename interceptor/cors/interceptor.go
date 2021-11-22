// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
//
// Package rkgrpccors is a CORS interceptor for grpc framework
package rkgrpccors

import (
	"net/http"
	"strconv"
	"strings"
)

func Interceptor(h http.Handler, opts ...Option) http.Handler {
	set := newOptionSet(opts...)

	allowMethods := strings.Join(set.AllowMethods, ",")
	allowHeaders := strings.Join(set.AllowHeaders, ",")
	exposeHeaders := strings.Join(set.ExposeHeaders, ",")
	maxAge := strconv.Itoa(set.MaxAge)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if set.Skipper(req) {
			h.ServeHTTP(w, req)
			return
		}

		originHeader := req.Header.Get(headerOrigin)
		preflight := req.Method == http.MethodOptions

		// 1: if no origin header was provided, we will return 204 if request is not a OPTION method
		if originHeader == "" {
			// 1.1: if not a preflight request, then pass through
			if !preflight {
				h.ServeHTTP(w, req)
				return
			}

			// 1.2: if it is a preflight request, then return with 204
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 2: origin not allowed, we will return 204 if request is not a OPTION method
		if !set.isOriginAllowed(originHeader) {
			// 2.1: if not a preflight request, then pass through
			if !preflight {
				w.WriteHeader(http.StatusFound)
				return
			}

			// 2.2: if it is a preflight request, then return with 204
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 3: not a OPTION method
		if !preflight {
			w.Header().Set(headerAccessControlAllowOrigin, originHeader)
			// 3.1: add Access-Control-Allow-Credentials
			if set.AllowCredentials {
				w.Header().Set(headerAccessControlAllowCredentials, "true")
			}
			// 3.2: add Access-Control-Expose-Headers
			if exposeHeaders != "" {
				w.Header().Set(headerAccessControlExposeHeaders, exposeHeaders)
			}
			h.ServeHTTP(w, req)
			return
		}

		// 4: preflight request, return 204
		// add related headers including:
		//
		// - Vary
		// - Access-Control-Allow-Origin
		// - Access-Control-Allow-Methods
		// - Access-Control-Allow-Credentials
		// - Access-Control-Allow-Headers
		// - Access-Control-Max-Age
		w.Header().Add(headerVary, headerAccessControlRequestMethod)
		w.Header().Add(headerVary, headerAccessControlRequestHeaders)
		w.Header().Set(headerAccessControlAllowOrigin, originHeader)
		w.Header().Set(headerAccessControlAllowMethods, allowMethods)

		// 4.1: Access-Control-Allow-Credentials
		if set.AllowCredentials {
			w.Header().Set(headerAccessControlAllowCredentials, "true")
		}

		// 4.2: Access-Control-Allow-Headers
		if allowHeaders != "" {
			w.Header().Set(headerAccessControlAllowHeaders, allowHeaders)
		} else {
			h := req.Header.Get(headerAccessControlRequestHeaders)
			if h != "" {
				w.Header().Set(headerAccessControlAllowHeaders, h)
			}
		}
		if set.MaxAge > 0 {
			// 4.3: Access-Control-Max-Age
			w.Header().Set(headerAccessControlMaxAge, maxAge)
		}

		w.WriteHeader(http.StatusNoContent)
		return
	})
}
