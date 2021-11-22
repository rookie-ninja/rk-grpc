// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpccors

import (
	"github.com/rookie-ninja/rk-grpc/interceptor"
	"net/http"
	"regexp"
	"strings"
)

const (
	headerOrigin                        = "Origin"
	headerAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	headerAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	headerAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	headerVary                          = "Vary"
	headerAccessControlRequestMethod    = "Access-Control-Request-Method"
	headerAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	headerAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	headerAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	headerAccessControlMaxAge           = "Access-Control-Max-Age"
)

// Interceptor would distinguish auth set based on.
var (
	optionsMap     = make(map[string]*optionSet)
	defaultSkipper = func(*http.Request) bool {
		return false
	}
)

// Create new optionSet with rpc type nad options.
func newOptionSet(opts ...Option) *optionSet {
	set := &optionSet{
		EntryName:        rkgrpcinter.RpcEntryNameValue,
		EntryType:        rkgrpcinter.RpcEntryTypeValue,
		Skipper:          defaultSkipper,
		AllowOrigins:     []string{},
		AllowMethods:     []string{},
		AllowHeaders:     []string{},
		AllowCredentials: false,
		ExposeHeaders:    []string{},
		MaxAge:           0,
	}

	for i := range opts {
		opts[i](set)
	}

	if len(set.AllowOrigins) < 1 {
		set.AllowOrigins = append(set.AllowOrigins, "*")
	}

	if len(set.AllowMethods) < 1 {
		set.AllowMethods = append(set.AllowMethods,
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete)
	}

	// parse regex pattern in origins
	set.toPatterns()

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
	// Skipper defines a function to skip middleware.
	Skipper Skipper
	// AllowOrigins defines a list of origins that may access the resource.
	// Optional. Default value []string{"*"}.
	AllowOrigins []string
	// allowPatterns derived from AllowOrigins by parsing regex fields
	// auto generated when creating new optionSet was created
	allowPatterns []string
	// AllowMethods defines a list methods allowed when accessing the resource.
	// This is used in response to a preflight request.
	// Optional. Default value DefaultCORSConfig.AllowMethods.
	AllowMethods []string
	// AllowHeaders defines a list of request headers that can be used when
	// making the actual request. This is in response to a preflight request.
	// Optional. Default value []string{}.
	AllowHeaders []string
	// AllowCredentials indicates whether or not the response to the request
	// can be exposed when the credentials flag is true. When used as part of
	// a response to a preflight request, this indicates whether or not the
	// actual request can be made using credentials.
	// Optional. Default value false.
	AllowCredentials bool
	// ExposeHeaders defines a whitelist headers that clients are allowed to
	// access.
	// Optional. Default value []string{}.
	ExposeHeaders []string
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached.
	// Optional. Default value 0.
	MaxAge int
}

// Convert allowed origins to patterns
func (set *optionSet) toPatterns() {
	set.allowPatterns = []string{}

	for _, raw := range set.AllowOrigins {
		var result strings.Builder
		result.WriteString("^")
		for i, literal := range strings.Split(raw, "*") {

			// Replace * with .*
			if i > 0 {
				result.WriteString(".*")
			}

			result.WriteString(literal)
		}
		result.WriteString("$")
		set.allowPatterns = append(set.allowPatterns, result.String())
	}
}

func (set *optionSet) isOriginAllowed(originHeader string) bool {
	res := false

	for _, pattern := range set.allowPatterns {
		res, _ = regexp.MatchString(pattern, originHeader)
		if res {
			break
		}
	}

	return res
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

// WithSkipper provide skipper.
func WithSkipper(skip Skipper) Option {
	return func(opt *optionSet) {
		opt.Skipper = skip
	}
}

// WithAllowOrigins provide allowed origins.
func WithAllowOrigins(origins ...string) Option {
	return func(opt *optionSet) {
		opt.AllowOrigins = append(opt.AllowOrigins, origins...)
	}
}

// WithAllowMethods provide allowed http methods
func WithAllowMethods(methods ...string) Option {
	return func(opt *optionSet) {
		opt.AllowMethods = append(opt.AllowMethods, methods...)
	}
}

// WithAllowHeaders provide allowed headers
func WithAllowHeaders(headers ...string) Option {
	return func(opt *optionSet) {
		opt.AllowHeaders = append(opt.AllowHeaders, headers...)
	}
}

// WithAllowCredentials allow credentials or not
func WithAllowCredentials(allow bool) Option {
	return func(opt *optionSet) {
		opt.AllowCredentials = allow
	}
}

// WithExposeHeaders provide expose headers
func WithExposeHeaders(headers ...string) Option {
	return func(opt *optionSet) {
		opt.ExposeHeaders = append(opt.ExposeHeaders, headers...)
	}
}

// WithMaxAge provide max age
func WithMaxAge(age int) Option {
	return func(opt *optionSet) {
		opt.MaxAge = age
	}
}

// Skipper default skipper will always return false
type Skipper func(*http.Request) bool
