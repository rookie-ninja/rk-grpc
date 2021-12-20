// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccsrf

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	// match 1
	handler := Interceptor(userHandler, WithSkipper(func(request *http.Request) bool {
		return true
	}))
	w, r := getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// match 2.1
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "_csrf")

	// match 2.2
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet)
	r.AddCookie(&http.Cookie{
		Name:  "_csrf",
		Value: "ut-csrf-token",
	})
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "_csrf")

	// match 3.1
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// match 3.2
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodPost)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// match 3.3
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodPost)
	r.Header.Set(headerXCSRFToken, "ut-csrf-token")
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// match 4.1
	handler = Interceptor(userHandler,
		WithCookiePath("ut-path"))
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "ut-path")

	// match 4.2
	handler = Interceptor(userHandler,
		WithCookieDomain("ut-domain"))
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "ut-domain")

	// match 4.3
	handler = Interceptor(userHandler,
		WithCookieSameSite("strict"))
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "Strict")
}
