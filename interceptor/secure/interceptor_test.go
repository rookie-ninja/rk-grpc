package rkgrpcsec

import (
	"github.com/rookie-ninja/rk-entry/middleware/secure"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	beforeCtx := rkmidsec.NewBeforeCtx()
	mock := rkmidsec.NewOptionSetMock(beforeCtx)

	// case 1: with error response
	inter := Interceptor(userHandler, rkmidsec.WithMockOptionSet(mock))
	req := httptest.NewRequest(http.MethodGet, "/ut", nil)
	w := httptest.NewRecorder()

	// assign any of error response
	beforeCtx.Output.HeadersToReturn["key"] = "value"
	inter.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "value", w.Header().Get("key"))
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
