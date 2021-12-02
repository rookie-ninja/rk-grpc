// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
//
// Package rkgrpcsec is a secure interceptor for grpc-gateway
package rkgrpcsec

import (
	"fmt"
	"net/http"
)

func Interceptor(h http.Handler, opts ...Option) http.Handler {
	set := newOptionSet(opts...)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if set.Skipper(req) {
			h.ServeHTTP(w, req)
			return
		}

		// Add X-XSS-Protection header
		if set.XSSProtection != "" {
			w.Header().Set(headerXXSSProtection, set.XSSProtection)
		}

		// Add X-Content-Type-Options header
		if set.ContentTypeNosniff != "" {
			w.Header().Set(headerXContentTypeOptions, set.ContentTypeNosniff)
		}

		// Add X-Frame-Options header
		if set.XFrameOptions != "" {
			w.Header().Set(headerXFrameOptions, set.XFrameOptions)
		}

		// Add Strict-Transport-Security header
		if (req.TLS != nil || (req.Header.Get(headerXForwardedProto) == "https")) && set.HSTSMaxAge != 0 {
			subdomains := ""
			if !set.HSTSExcludeSubdomains {
				subdomains = "; includeSubdomains"
			}
			if set.HSTSPreloadEnabled {
				subdomains = fmt.Sprintf("%s; preload", subdomains)
			}
			w.Header().Set(headerStrictTransportSecurity, fmt.Sprintf("max-age=%d%s", set.HSTSMaxAge, subdomains))
		}

		// Add Content-Security-Policy-Report-Only or Content-Security-Policy header
		if set.ContentSecurityPolicy != "" {
			if set.CSPReportOnly {
				w.Header().Set(headerContentSecurityPolicyReportOnly, set.ContentSecurityPolicy)
			} else {
				w.Header().Set(headerContentSecurityPolicy, set.ContentSecurityPolicy)
			}
		}

		// Add Referrer-Policy header
		if set.ReferrerPolicy != "" {
			w.Header().Set(headerReferrerPolicy, set.ReferrerPolicy)
		}

		h.ServeHTTP(w, req)
	})
}
