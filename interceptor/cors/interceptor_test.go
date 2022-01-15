// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccors

import (
	"github.com/rookie-ninja/rk-entry/middleware"
	"github.com/rookie-ninja/rk-entry/middleware/cors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	beforeCtx := rkmidcors.NewBeforeCtx()
	beforeCtx.Output.HeadersToReturn["key"] = "value"
	beforeCtx.Output.HeaderVary = []string{"vary"}
	mock := rkmidcors.NewOptionSetMock(beforeCtx)

	// case 1: abort
	inter := Interceptor(userHandler, rkmidcors.WithMockOptionSet(mock))
	beforeCtx.Output.Abort = true
	req := httptest.NewRequest(http.MethodGet, "/ut", nil)
	w := httptest.NewRecorder()
	inter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "value", w.Header().Get("key"))
	assert.Equal(t, "vary", w.Header().Get(rkmid.HeaderVary))

	// case 2: happy case
	beforeCtx.Output.Abort = false
	req = httptest.NewRequest(http.MethodGet, "/ut", nil)
	w = httptest.NewRecorder()
	inter.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ************ Test utility ************

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
