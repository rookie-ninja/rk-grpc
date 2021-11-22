// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccors

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

const originHeaderValue = "http://ut-origin"

func assertNotPanic(t *testing.T) {
	if r := recover(); r != nil {
		// Expect panic to be called with non nil error
		assert.True(t, false)
	} else {
		// This should never be called in case of a bug
		assert.True(t, true)
	}
}

type user struct{}

func (h *user) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

var userHandler *user

func TestInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	// with skipper
	handler := Interceptor(userHandler, WithSkipper(func(request *http.Request) bool {
		return true
	}))
	w, r := getReqAndResp(http.MethodGet, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// with empty option, all request will be passed
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// match 1.1
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// match 1.2
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodOptions)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// match 2.1
	handler = Interceptor(userHandler, WithAllowOrigins("http://do-not-pass-through"))
	w, r = getReqAndResp(http.MethodGet, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusFound, w.Code)

	// match 2.2
	handler = Interceptor(userHandler, WithAllowOrigins("http://do-not-pass-through"))
	w, r = getReqAndResp(http.MethodOptions, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// match 3
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))

	// match 3.1
	handler = Interceptor(userHandler, WithAllowCredentials(true))
	w, r = getReqAndResp(http.MethodGet, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))
	assert.Equal(t, "true", w.Header().Get(headerAccessControlAllowCredentials))

	// match 3.2
	handler = Interceptor(userHandler,
		WithAllowCredentials(true),
		WithExposeHeaders("expose"))
	w, r = getReqAndResp(http.MethodGet, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))
	assert.Equal(t, "true", w.Header().Get(headerAccessControlAllowCredentials))
	assert.Equal(t, "expose", w.Header().Get(headerAccessControlExposeHeaders))

	// match 4
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodOptions, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{
		headerAccessControlRequestMethod,
		headerAccessControlRequestHeaders,
	}, w.Header().Values(headerVary))
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))

	// match 4.1
	handler = Interceptor(userHandler, WithAllowCredentials(true))
	w, r = getReqAndResp(http.MethodOptions, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{
		headerAccessControlRequestMethod,
		headerAccessControlRequestHeaders,
	}, w.Header().Values(headerVary))
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))
	assert.NotEmpty(t, w.Header().Get(headerAccessControlAllowMethods))
	assert.Equal(t, "true", w.Header().Get(headerAccessControlAllowCredentials))

	// match 4.2
	handler = Interceptor(userHandler, WithAllowHeaders("ut-header"))
	w, r = getReqAndResp(http.MethodOptions, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{
		headerAccessControlRequestMethod,
		headerAccessControlRequestHeaders,
	}, w.Header().Values(headerVary))
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))
	assert.NotEmpty(t, w.Header().Get(headerAccessControlAllowMethods))
	assert.Equal(t, "ut-header", w.Header().Get(headerAccessControlAllowHeaders))

	// match 4.3
	handler = Interceptor(userHandler, WithMaxAge(1))
	w, r = getReqAndResp(http.MethodOptions, header{headerOrigin, originHeaderValue})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{
		headerAccessControlRequestMethod,
		headerAccessControlRequestHeaders,
	}, w.Header().Values(headerVary))
	assert.Equal(t, originHeaderValue, w.Header().Get(headerAccessControlAllowOrigin))
	assert.NotEmpty(t, w.Header().Get(headerAccessControlAllowMethods))
	assert.Equal(t, "1", w.Header().Get(headerAccessControlMaxAge))
}

func getReqAndResp(method string, headers ...header) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, "/get", nil)
	for _, h := range headers {
		req.Header.Add(h.Key, h.Value)
	}
	w := httptest.NewRecorder()
	return w, req
}

type header struct {
	Key   string
	Value string
}
