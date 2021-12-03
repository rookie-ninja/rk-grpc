// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
//
// Package rkgrpccsrf is a CSRF interceptor for grpc framework
package rkgrpccsrf

import (
	"encoding/json"
	rkcommon "github.com/rookie-ninja/rk-common/common"
	rkerror "github.com/rookie-ninja/rk-common/error"
	rkgrpcinter "github.com/rookie-ninja/rk-grpc/interceptor"
	"net/http"
	"time"
)

func Interceptor(h http.Handler, opts ...Option) http.Handler {
	set := newOptionSet(opts...)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if set.Skipper(req) {
			h.ServeHTTP(w, req)
			return
		}

		k, err := req.Cookie(set.CookieName)
		token := ""

		// 2.1: generate token if failed to get cookie from context
		if err != nil {
			token = rkcommon.RandString(set.TokenLength)
		} else {
			// 2.2: reuse token if exists
			token = k.Value
		}

		// 3.1: do not check http methods of GET, HEAD, OPTIONS and TRACE
		switch req.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		default:
			// 3.2: validate token only for requests which are not defined as 'safe' by RFC7231
			clientToken, err := set.extractor(req)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				bytes, _ := json.Marshal(rkerror.New(
					rkerror.WithHttpCode(http.StatusBadRequest),
					rkerror.WithMessage("failed to extract client token"),
					rkerror.WithDetails(err)))
				w.Write(bytes)
				return
			}

			// 3.3: return 403 to client if token is not matched
			if !isValidToken(token, clientToken) {
				w.WriteHeader(http.StatusForbidden)
				bytes, _ := json.Marshal(rkerror.New(
					rkerror.WithHttpCode(http.StatusForbidden),
					rkerror.WithMessage("invalid csrf token"),
					rkerror.WithDetails(err)))
				w.Write(bytes)
				return
			}
		}

		// set CSRF cookie
		cookie := new(http.Cookie)
		cookie.Name = set.CookieName
		cookie.Value = token
		// 4.1
		if set.CookiePath != "" {
			cookie.Path = set.CookiePath
		}
		// 4.2
		if set.CookieDomain != "" {
			cookie.Domain = set.CookieDomain
		}
		// 4.3
		if set.CookieSameSite != http.SameSiteDefaultMode {
			cookie.SameSite = set.CookieSameSite
		}
		cookie.Expires = time.Now().Add(time.Duration(set.CookieMaxAge) * time.Second)
		cookie.Secure = set.CookieSameSite == http.SameSiteNoneMode
		cookie.HttpOnly = set.CookieHTTPOnly
		http.SetCookie(w, cookie)

		// store token in the request header
		req.Header.Set(rkgrpcinter.RpcCsrfTokenKey, token)

		// protect clients from caching the response
		w.Header().Add(headerVary, headerCookie)

		h.ServeHTTP(w, req)
	})
}
