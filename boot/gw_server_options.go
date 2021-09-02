// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package rkgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rookie-ninja/rk-common/error"
	"github.com/rookie-ninja/rk-entry/entry"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"net/textproto"
	"strings"
)

var (
	RkGwServerMuxOptions = []runtime.ServeMuxOption{
		runtime.WithErrorHandler(HttpErrorHandler),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   false,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{},
		}),
		runtime.WithMetadata(func(c context.Context, req *http.Request) metadata.MD {
			// we are unable to get scheme with req.URL.Scheme.
			// Let's check with TLS.
			scheme := "http"
			if req.TLS != nil {
				scheme = "https"
			}

			return metadata.Pairs(
				"x-forwarded-method", req.Method,
				"x-forwarded-path", req.URL.Path,
				"x-forwarded-scheme", scheme,
				"x-forwarded-user-agent", req.UserAgent(),
				"x-forwarded-remote-addr", req.RemoteAddr)
		}),
		runtime.WithOutgoingHeaderMatcher(OutgoingHeaderMatcher),
		runtime.WithIncomingHeaderMatcher(IncomingHeaderMatcher),
	}
)

// Mainly copies from runtime.DefaultHTTPErrorHandler.
// We reformat error response with rkerror.ErrorResp.
func HttpErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	logger := rkentry.GlobalAppCtx.GetZapLoggerEntryDefault().GetLogger()

	s := status.Convert(err)
	pb := s.Proto()

	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")

	contentType := marshaler.ContentType(pb)
	w.Header().Set("Content-Type", contentType)

	resp := rkerror.New()
	resp.Err.Code = runtime.HTTPStatusFromCode(s.Code())
	resp.Err.Message = s.Message()
	resp.Err.Status = http.StatusText(resp.Err.Code)
	resp.Err.Details = append(resp.Err.Details, s.Details()...)

	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		logger.Warn("Failed to extract ServerMetadata from context")
	}

	// handle forward response server metadata
	for k, vs := range md.HeaderMD {
		if h, ok := OutgoingHeaderMatcher(k); ok {
			for _, v := range vs {
				w.Header().Add(h, v)
			}
		}
	}

	// RFC 7230 https://tools.ietf.org/html/rfc7230#section-4.1.2
	// Unless the request includes a TE header field indicating "trailers"
	// is acceptable, as described in Section 4.3, a server SHOULD NOT
	// generate trailer fields that it believes are necessary for the user
	// agent to receive.
	var wantsTrailers bool

	if te := r.Header.Get("TE"); strings.Contains(strings.ToLower(te), "trailers") {
		wantsTrailers = true
		// handle forward response trailer header
		for k := range md.TrailerMD {
			tKey := textproto.CanonicalMIMEHeaderKey(fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k))
			w.Header().Add("Trailer", tKey)
		}

		w.Header().Set("Transfer-Encoding", "chunked")
	}

	st := runtime.HTTPStatusFromCode(s.Code())
	w.WriteHeader(st)

	bytes, _ := json.Marshal(resp)

	if _, err := w.Write(bytes); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}

	if wantsTrailers {
		// handle forward response trailer
		for k, vs := range md.TrailerMD {
			tKey := fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k)
			for _, v := range vs {
				w.Header().Add(tKey, v)
			}
		}
	}
}

// Pass out all metadata in grpc to http header.
func OutgoingHeaderMatcher(key string) (string, bool) {
	return key, true
}

// Pass out all metadata in http header to grpc metadata.
func IncomingHeaderMatcher(key string) (string, bool) {
	return key, true
}
