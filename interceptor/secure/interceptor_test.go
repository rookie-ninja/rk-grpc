package rkgrpcsec

import (
	"crypto/tls"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInterceptor(t *testing.T) {
	defer assertNotPanic(t)

	// with skipper
	handler := Interceptor(userHandler, WithSkipper(func(request *http.Request) bool {
		return true
	}))
	w, r := getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// without options
	handler = Interceptor(userHandler)
	w, r = getReqAndResp(http.MethodGet)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	containsHeader(t, w,
		headerXXSSProtection,
		headerXContentTypeOptions,
		headerXFrameOptions)

	// with options
	handler = Interceptor(userHandler,
		WithXSSProtection("ut-xss"),
		WithContentTypeNosniff("ut-sniff"),
		WithXFrameOptions("ut-frame"),
		WithHSTSMaxAge(10),
		WithHSTSExcludeSubdomains(true),
		WithHSTSPreloadEnabled(true),
		WithContentSecurityPolicy("ut-policy"),
		WithCSPReportOnly(true),
		WithReferrerPolicy("ut-ref"),
		WithIgnorePrefix("ut-prefix"))
	w, r = getReqAndResp(http.MethodGet)
	r.TLS = &tls.ConnectionState{}
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	containsHeader(t, w,
		headerXXSSProtection,
		headerXContentTypeOptions,
		headerXFrameOptions,
		headerStrictTransportSecurity,
		headerContentSecurityPolicyReportOnly,
		headerReferrerPolicy)
}

func containsHeader(t *testing.T, w http.ResponseWriter, headers ...string) {
	for _, v := range headers {
		assert.Contains(t, w.Header(), v)
	}
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
