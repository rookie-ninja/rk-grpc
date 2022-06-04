// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpccsrf is a CSRF interceptor for grpc framework
package rkgrpccsrf

import (
	"encoding/json"
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/csrf"
	"net/http"
)

func Interceptor(h http.Handler, opts ...rkmidcsrf.Option) http.Handler {
	set := rkmidcsrf.NewOptionSet(opts...)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		beforeCtx := set.BeforeCtx(req)
		set.Before(beforeCtx)

		if beforeCtx.Output.ErrResp != nil {
			w.WriteHeader(beforeCtx.Output.ErrResp.Code())
			bytes, _ := json.Marshal(beforeCtx.Output.ErrResp)
			w.Write(bytes)
			return
		}

		for _, v := range beforeCtx.Output.VaryHeaders {
			w.Header().Add(rkmid.HeaderVary, v)
		}

		if beforeCtx.Output.Cookie != nil {
			http.SetCookie(w, beforeCtx.Output.Cookie)
		}

		req.Header.Set(rkmid.CsrfTokenKey.String(), beforeCtx.Input.Token)

		h.ServeHTTP(w, req)
	})
}
