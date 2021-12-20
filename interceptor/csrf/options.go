// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccsrf

import (
	"crypto/subtle"
	"errors"
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
	headerXCSRFToken = "X-CSRF-Token"
	headerCookie     = "Cookie"
	headerVary       = "Vary"
)

// Skipper default skipper will always return false
type Skipper func(*http.Request) bool

// CsrfTokenExtractor defines a function that takes `echo.Context` and returns
// either a token or an error.
type csrfTokenExtractor func(*http.Request) (string, error)

// Create new optionSet with rpc type nad options.
func newOptionSet(opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:      rkgrpcinter.RpcEntryNameValue,
		EntryType:      rkgrpcinter.RpcEntryTypeValue,
		Skipper:        defaultSkipper,
		TokenLength:    32,
		TokenLookup:    "header:" + headerXCSRFToken,
		CookieName:     "_csrf",
		CookieMaxAge:   86400,
		CookieSameSite: http.SameSiteDefaultMode,
		IgnorePrefix:   make([]string, 0),
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

	// initialize extractor
	parts := strings.Split(set.TokenLookup, ":")
	set.extractor = csrfTokenFromHeader(parts[1])
	switch parts[0] {
	case "form":
		set.extractor = csrfTokenFromForm(parts[1])
	case "query":
		set.extractor = csrfTokenFromQuery(parts[1])
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

	// TokenLength is the length of the generated token.
	TokenLength int

	// TokenLookup is a string in the form of "<source>:<key>" that is used
	// to extract token from the request.
	// Optional. Default value "header:X-CSRF-Token".
	// Possible values:
	// - "header:<name>"
	// - "form:<name>"
	// - "query:<name>"
	TokenLookup string

	// CookieName Name of the CSRF cookie. This cookie will store CSRF token.
	// Optional. Default value "_csrf".
	CookieName string

	// CookieDomain Domain of the CSRF cookie.
	// Optional. Default value none.
	CookieDomain string

	// CookiePath Path of the CSRF cookie.
	// Optional. Default value none.
	CookiePath string

	// CookieMaxAge Max age (in seconds) of the CSRF cookie.
	// Optional. Default value 86400 (24hr).
	CookieMaxAge int

	// CookieHTTPOnly Indicates if CSRF cookie is HTTP only.
	// Optional. Default value false.
	CookieHTTPOnly bool

	// CookieSameSite Indicates SameSite mode of the CSRF cookie.
	// Optional. Default value SameSiteDefaultMode.
	CookieSameSite http.SameSite

	extractor csrfTokenExtractor
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

// WithTokenLength the length of the generated token.
// Optional. Default value 32.
func WithTokenLength(val int) Option {
	return func(opt *optionSet) {
		if val > 0 {
			opt.TokenLength = val
		}
	}
}

// WithTokenLookup a string in the form of "<source>:<key>" that is used
// to extract token from the request.
// Optional. Default value "header:X-CSRF-Token".
// Possible values:
// - "header:<name>"
// - "form:<name>"
// - "query:<name>"
// Optional. Default value "header:X-CSRF-Token".
func WithTokenLookup(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.TokenLookup = val
		}
	}
}

// WithCookieName provide name of the CSRF cookie. This cookie will store CSRF token.
// Optional. Default value "csrf".
func WithCookieName(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.CookieName = val
		}
	}
}

// WithCookieDomain provide domain of the CSRF cookie.
// Optional. Default value "".
func WithCookieDomain(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.CookieDomain = val
		}
	}
}

// WithCookiePath provide path of the CSRF cookie.
// Optional. Default value "".
func WithCookiePath(val string) Option {
	return func(opt *optionSet) {
		if len(val) > 0 {
			opt.CookiePath = val
		}
	}
}

// WithCookieMaxAge provide max age (in seconds) of the CSRF cookie.
// Optional. Default value 86400 (24hr).
func WithCookieMaxAge(val int) Option {
	return func(opt *optionSet) {
		if val > 0 {
			opt.CookieMaxAge = val
		}
	}
}

// WithCookieHTTPOnly indicates if CSRF cookie is HTTP only.
// Optional. Default value false.
func WithCookieHTTPOnly(val bool) Option {
	return func(opt *optionSet) {
		opt.CookieHTTPOnly = val
	}
}

// WithCookieSameSite indicates SameSite mode of the CSRF cookie.
// Optional. Default value SameSiteDefaultMode.
func WithCookieSameSite(val string) Option {
	return func(opt *optionSet) {
		val = strings.ToLower(val)
		switch val {
		case "lax":
			opt.CookieSameSite = http.SameSiteLaxMode
		case "strict":
			opt.CookieSameSite = http.SameSiteStrictMode
		case "none":
			opt.CookieSameSite = http.SameSiteNoneMode
		default:
			opt.CookieSameSite = http.SameSiteDefaultMode
		}
	}
}

// WithIgnorePrefix provide paths prefix that will ignore.
// Mainly used for swagger main page and RK TV entry.
func WithIgnorePrefix(paths ...string) Option {
	return func(set *optionSet) {
		set.IgnorePrefix = append(set.IgnorePrefix, paths...)
	}
}

// csrfTokenFromForm returns a `csrfTokenExtractor` that extracts token from the
// provided request header.
func csrfTokenFromHeader(header string) csrfTokenExtractor {
	return func(req *http.Request) (string, error) {
		token := req.Header.Get(header)
		if token == "" {
			return "", errors.New("missing csrf token in header")
		}
		return token, nil
	}
}

// csrfTokenFromForm returns a `csrfTokenExtractor` that extracts token from the
// provided form parameter.
func csrfTokenFromForm(param string) csrfTokenExtractor {
	return func(req *http.Request) (string, error) {
		token := req.FormValue(param)
		if token == "" {
			return "", errors.New("missing csrf token in the form parameter")
		}
		return token, nil
	}
}

// csrfTokenFromQuery returns a `csrfTokenExtractor` that extracts token from the
// provided query parameter.
func csrfTokenFromQuery(param string) csrfTokenExtractor {
	return func(req *http.Request) (string, error) {
		token := req.URL.Query().Get(param)
		if token == "" {
			return "", errors.New("missing csrf token in the query string")
		}
		return token, nil
	}
}

func isValidToken(token, clientToken string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(clientToken)) == 1
}
