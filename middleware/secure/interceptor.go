// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
//
// Package rkgrpcsec is a secure interceptor for grpc-gateway

package rkgrpcsec

import (
	"github.com/rookie-ninja/rk-entry/v2/middleware/secure"
	"net/http"
)

func Interceptor(h http.Handler, opts ...rkmidsec.Option) http.Handler {
	set := rkmidsec.NewOptionSet(opts...)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// case 1: return to user if error occur
		beforeCtx := set.BeforeCtx(req)
		set.Before(beforeCtx)

		for k, v := range beforeCtx.Output.HeadersToReturn {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, req)
	})
}
