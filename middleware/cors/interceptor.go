// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
//
// Package rkgrpccors is a CORS interceptor for grpc framework

package rkgrpccors

import (
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/cors"
	"net/http"
)

func Interceptor(h http.Handler, opts ...rkmidcors.Option) http.Handler {
	set := rkmidcors.NewOptionSet(opts...)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		beforeCtx := set.BeforeCtx(req)
		set.Before(beforeCtx)

		for k, v := range beforeCtx.Output.HeadersToReturn {
			w.Header().Set(k, v)
		}

		for _, v := range beforeCtx.Output.HeaderVary {
			w.Header().Add(rkmid.HeaderVary, v)
		}

		// case 1: with abort
		if beforeCtx.Output.Abort {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// case 2: call next
		h.ServeHTTP(w, req)

		return
	})
}
