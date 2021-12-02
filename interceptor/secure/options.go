// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcsec

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"net/http"
	"reflect"
	"strings"
)

// Interceptor would distinguish auth set based on.
var (
	optionsMap     = make(map[string]*optionSet)
	defaultSkipper = func(r *http.Request) bool {
		return false
	}
)

const (
	headerXXSSProtection                  = "X-Xss-Protection"
	headerXContentTypeOptions             = "X-Content-Type-Options"
	headerXFrameOptions                   = "X-Frame-Options"
	headerXForwardedProto                 = "X-Forwarded-Proto"
	headerStrictTransportSecurity         = "Strict-Transport-Security"
	headerContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	headerContentSecurityPolicy           = "Content-Security-Policy"
	headerReferrerPolicy                  = "Referrer-Policy"
)

// Skipper default skipper will always return false
type Skipper func(r *http.Request) bool

// Create new optionSet with rpc type nad options.
func newOptionSet(opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:          rkgrpcinter.RpcEntryNameValue,
		EntryType:          rkgrpcinter.RpcEntryTypeValue,
		Skipper:            defaultSkipper,
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
		HSTSPreloadEnabled: false,
		IgnorePrefix:       make([]string, 0),
	}

	for i := range opts {
		opts[i](set)
	}

	// default skipper was used, override it with ignoring prefix
	if reflect.ValueOf(set.Skipper).Pointer() == reflect.ValueOf(defaultSkipper).Pointer() {
		set.Skipper = func(req *http.Request) bool {
			if req == nil {
				return false
			}

			urlPath := req.URL.Path

			for i := range set.IgnorePrefix {
				if strings.HasPrefix(urlPath, set.IgnorePrefix[i]) {
					return true
				}
			}

			return false
		}
	}

	if _, ok := optionsMap[set.EntryName]; !ok {
		optionsMap[set.EntryName] = set
	}

	return set
}

// Options which is used while initializing extension interceptor
type optionSet struct {
	// EntryName name of entry
	EntryName string

	// EntryType type of entry
	EntryType string

	// Skipper function
	Skipper Skipper

	// IgnorePrefix ignoring paths prefix
	IgnorePrefix []string

	// XSSProtection provides protection against cross-site scripting attack (XSS)
	// by setting the `X-XSS-Protection` header.
	// Optional. Default value "1; mode=block".
	XSSProtection string

	// ContentTypeNosniff provides protection against overriding Content-Type
	// header by setting the `X-Content-Type-Options` header.
	// Optional. Default value "nosniff".
	ContentTypeNosniff string

	// XFrameOptions can be used to indicate whether or not a browser should
	// be allowed to render a page in a <frame>, <iframe> or <object> .
	// Sites can use this to avoid clickjacking attacks, by ensuring that their
	// content is not embedded into other sites.provides protection against
	// clickjacking.
	// Optional. Default value "SAMEORIGIN".
	// Possible values:
	// - "SAMEORIGIN" - The page can only be displayed in a frame on the same origin as the page itself.
	// - "DENY" - The page cannot be displayed in a frame, regardless of the site attempting to do so.
	// - "ALLOW-FROM uri" - The page can only be displayed in a frame on the specified origin.
	XFrameOptions string

	// HSTSMaxAge sets the `Strict-Transport-Security` header to indicate how
	// long (in seconds) browsers should remember that this site is only to
	// be accessed using HTTPS. This reduces your exposure to some SSL-stripping
	// man-in-the-middle (MITM) attacks.
	// Optional. Default value 0.
	HSTSMaxAge int

	// HSTSExcludeSubdomains won't include subdomains tag in the `Strict Transport Security`
	// header, excluding all subdomains from security policy. It has no effect
	// unless HSTSMaxAge is set to a non-zero value.
	// Optional. Default value false.
	HSTSExcludeSubdomains bool

	// ContentSecurityPolicy sets the `Content-Security-Policy` header providing
	// security against cross-site scripting (XSS), clickjacking and other code
	// injection attacks resulting from execution of malicious content in the
	// trusted web page context.
	// Optional. Default value "".
	ContentSecurityPolicy string

	// HSTSPreloadEnabled will add the preload tag in the `Strict Transport Security`
	// header, which enables the domain to be included in the HSTS preload list
	// maintained by Chrome (and used by Firefox and Safari): https://hstspreload.org/
	// Optional.  Default value false.
	HSTSPreloadEnabled bool

	// CSPReportOnly would use the `Content-Security-Policy-Report-Only` header instead
	// of the `Content-Security-Policy` header. This allows iterative updates of the
	// content security policy by only reporting the violations that would
	// have occurred instead of blocking the resource.
	// Optional. Default value false.
	CSPReportOnly bool

	// ReferrerPolicy sets the `Referrer-Policy` header providing security against
	// leaking potentially sensitive request paths to third parties.
	// Optional. Default value "".
	ReferrerPolicy string
}

// Option if for middleware options while creating middleware
type Option func(*optionSet)

// WithEntryNameAndType provide entry name and entry type.
func WithEntryNameAndType(entryName, entryType string) Option {
	return func(opt *optionSet) {
		opt.EntryName = entryName
		opt.EntryType = entryType
	}
}

// WithSkipper provide Skipper.
func WithSkipper(skip Skipper) Option {
	return func(opt *optionSet) {
		opt.Skipper = skip
	}
}

// WithXSSProtection provide X-XSS-Protection header value.
// Optional. Default value "1; mode=block".
func WithXSSProtection(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.XSSProtection = val
		}
	}
}

// WithContentTypeNosniff provide X-Content-Type-Options header value.
// Optional. Default value "nosniff".
func WithContentTypeNosniff(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.ContentTypeNosniff = val
		}
	}
}

// WithXFrameOptions provide X-Frame-Options header value.
// Optional. Default value "SAMEORIGIN".
func WithXFrameOptions(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.XFrameOptions = val
		}
	}
}

// WithHSTSMaxAge provide Strict-Transport-Security header value.
func WithHSTSMaxAge(val int) Option {
	return func(opt *optionSet) {
		opt.HSTSMaxAge = val
	}
}

// WithHSTSExcludeSubdomains provide excluding subdomains of HSTS.
func WithHSTSExcludeSubdomains(val bool) Option {
	return func(opt *optionSet) {
		opt.HSTSExcludeSubdomains = val
	}
}

// WithHSTSPreloadEnabled provide enabling HSTS preload.
// Optional. Default value false.
func WithHSTSPreloadEnabled(val bool) Option {
	return func(opt *optionSet) {
		opt.HSTSPreloadEnabled = val
	}
}

// WithContentSecurityPolicy provide Content-Security-Policy header value.
// Optional. Default value "".
func WithContentSecurityPolicy(val string) Option {
	return func(opt *optionSet) {
		opt.ContentSecurityPolicy = val
	}
}

// WithCSPReportOnly provide Content-Security-Policy-Report-Only header value.
// Optional. Default value false.
func WithCSPReportOnly(val bool) Option {
	return func(set *optionSet) {
		set.CSPReportOnly = val
	}
}

// WithReferrerPolicy provide Referrer-Policy header value.
// Optional. Default value "".
func WithReferrerPolicy(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.ReferrerPolicy = val
		}
	}
}

// WithIgnorePrefix ignoring paths by prefix.
func WithIgnorePrefix(prefix ...string) Option {
	return func(opt *optionSet) {
		opt.IgnorePrefix = append(opt.IgnorePrefix, prefix...)
	}
}
