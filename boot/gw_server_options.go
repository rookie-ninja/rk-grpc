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
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"net/textproto"
	"strings"
)

// Gateway options for marshaller and unmarshaler.
//
// The inner fields was defined as pointer instead of reference which look strange.
//
// It is because we hope to make sure the value was defined by user in YAML file.
// Otherwise, the boolean value will always be false even there nothing in YAML file.
type gwOption struct {
	Marshal *struct {
		Multiline       *bool   `yaml:"multiline" json:"multiline"`
		EmitUnpopulated *bool   `yaml:"emitUnpopulated" json:"emitUnpopulated"`
		Indent          *string `yaml:"indent" json:"indent"`
		AllowPartial    *bool   `yaml:"allowPartial" json:"allowPartial"`
		UseProtoNames   *bool   `yaml:"useProtoNames" json:"useProtoNames"`
		UseEnumNumbers  *bool   `yaml:"useEnumNumbers" json:"useEnumNumbers"`
	} `yaml:"marshal" json:"marshal"`
	Unmarshal *struct {
		AllowPartial   *bool `yaml:"allowPartial" json:"allowPartial"`
		DiscardUnknown *bool `yaml:"discardUnknown" json:"discardUnknown"`
	} `yaml:"unmarshal" json:"unmarshal"`
}

// Convert gwOption to protojson.MarshalOptions
func toMarshalOptions(opt *gwOption) *protojson.MarshalOptions {
	res := &protojson.MarshalOptions{}

	if opt == nil || opt.Marshal == nil {
		return res
	}

	// Parse fields on by one
	if opt.Marshal.Multiline != nil {
		res.Multiline = *opt.Marshal.Multiline
	}
	if opt.Marshal.EmitUnpopulated != nil {
		res.EmitUnpopulated = *opt.Marshal.EmitUnpopulated
	}
	if opt.Marshal.Indent != nil {
		res.Indent = *opt.Marshal.Indent
	}
	if opt.Marshal.AllowPartial != nil {
		res.AllowPartial = *opt.Marshal.AllowPartial
	}
	if opt.Marshal.UseProtoNames != nil {
		res.UseProtoNames = *opt.Marshal.UseProtoNames
	}
	if opt.Marshal.UseEnumNumbers != nil {
		res.UseEnumNumbers = *opt.Marshal.UseEnumNumbers
	}

	return res
}

// Convert gwOption to protojson.UnmarshalOptions
func toUnmarshalOptions(opt *gwOption) *protojson.UnmarshalOptions {
	res := &protojson.UnmarshalOptions{}

	if opt == nil || opt.Unmarshal == nil {
		return res
	}

	if opt.Unmarshal.AllowPartial != nil {
		res.AllowPartial = *opt.Unmarshal.AllowPartial
	}
	if opt.Unmarshal.DiscardUnknown != nil {
		res.DiscardUnknown = *opt.Unmarshal.DiscardUnknown
	}

	return res
}

// Merge gwOption with default RK style protojson.MarshalOptions
func mergeWithRkGwMarshalOption(opt *gwOption) *protojson.MarshalOptions {
	res := &protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: true,
	}

	if opt == nil || opt.Marshal == nil {
		return res
	}

	// Parse fields on by one
	if opt.Marshal.Multiline != nil {
		res.Multiline = *opt.Marshal.Multiline
	}
	if opt.Marshal.EmitUnpopulated != nil {
		res.EmitUnpopulated = *opt.Marshal.EmitUnpopulated
	}
	if opt.Marshal.Indent != nil {
		res.Indent = *opt.Marshal.Indent
	}
	if opt.Marshal.AllowPartial != nil {
		res.AllowPartial = *opt.Marshal.AllowPartial
	}
	if opt.Marshal.UseProtoNames != nil {
		res.UseProtoNames = *opt.Marshal.UseProtoNames
	}
	if opt.Marshal.UseEnumNumbers != nil {
		res.UseEnumNumbers = *opt.Marshal.UseEnumNumbers
	}

	return res
}

// Merge gwOption with default RK style protojson.UnmarshalOptions
func mergeWithRkGwUnmarshalOption(opt *gwOption) *protojson.UnmarshalOptions {
	res := &protojson.UnmarshalOptions{}

	if opt == nil || opt.Unmarshal == nil {
		return res
	}

	// Parse fields on by one
	if opt.Unmarshal.AllowPartial != nil {
		res.AllowPartial = *opt.Unmarshal.AllowPartial
	}
	if opt.Unmarshal.DiscardUnknown != nil {
		res.DiscardUnknown = *opt.Unmarshal.DiscardUnknown
	}

	return res
}

// NewRkGwServerMuxOptions creates new gw server mux options.
func NewRkGwServerMuxOptions(mOptIn *protojson.MarshalOptions, uOptIn *protojson.UnmarshalOptions) []runtime.ServeMuxOption {
	mOpt := &protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: true,
	}

	if mOptIn != nil {
		mOpt = mOptIn
	}

	uOpt := &protojson.UnmarshalOptions{}
	if uOptIn != nil {
		uOpt = uOptIn
	}

	return []runtime.ServeMuxOption{
		runtime.WithErrorHandler(HttpErrorHandler),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   *mOpt,
			UnmarshalOptions: *uOpt,
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
}

// HttpErrorHandler Mainly copies from runtime.DefaultHTTPErrorHandler.
// We reformat error response with rkerror.ErrorResp.
func HttpErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
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

	md, _ := runtime.ServerMetadataFromContext(ctx)

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

// OutgoingHeaderMatcher Pass out all metadata in grpc to http header.
func OutgoingHeaderMatcher(key string) (string, bool) {
	return key, true
}

// IncomingHeaderMatcher Pass out all metadata in http header to grpc metadata.
func IncomingHeaderMatcher(key string) (string, bool) {
	return key, true
}
