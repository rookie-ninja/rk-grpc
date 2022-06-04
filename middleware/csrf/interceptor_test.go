// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccsrf

import (
	"github.com/rookie-ninja/rk-entry/v2/middleware"
	"github.com/rookie-ninja/rk-entry/v2/middleware/csrf"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	beforeCtx := rkmidcsrf.NewBeforeCtx()
	mock := rkmidcsrf.NewOptionSetMock(beforeCtx)

	// case 1: with error response
	inter := Interceptor(userHandler, rkmidcsrf.WithMockOptionSet(mock))
	req := httptest.NewRequest(http.MethodGet, "/ut", nil)
	w := httptest.NewRecorder()

	// assign any of error response
	beforeCtx.Output.ErrResp = rkmid.GetErrorBuilder().New(http.StatusForbidden, "")
	inter.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// case 2: happy case
	beforeCtx.Output.ErrResp = nil
	beforeCtx.Output.VaryHeaders = []string{"value"}
	beforeCtx.Output.Cookie = &http.Cookie{}
	req = httptest.NewRequest(http.MethodGet, "/ut", nil)
	w = httptest.NewRecorder()
	inter.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(rkmid.HeaderVary))
	assert.NotNil(t, w.Header().Get("Set-Cookie"))
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
