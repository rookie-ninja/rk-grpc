// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcerr

import (
	"fmt"
	"github.com/rookie-ninja/rk-grpc/boot/error/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorWrapper will wrap grpc error into rk style error
type ErrorWrapper func(msg string, errors ...error) *status.Status

var (
	Canceled           = BaseErrorWrapper(codes.Canceled)           // Canceled as name mentioned
	Unknown            = BaseErrorWrapper(codes.Unknown)            // Unknown as name mentioned
	InvalidArgument    = BaseErrorWrapper(codes.InvalidArgument)    // InvalidArgument as name mentioned
	DeadlineExceeded   = BaseErrorWrapper(codes.DeadlineExceeded)   // DeadlineExceeded as name mentioned
	NotFound           = BaseErrorWrapper(codes.NotFound)           // NotFound as name mentioned
	AlreadyExists      = BaseErrorWrapper(codes.AlreadyExists)      // AlreadyExists as name mentioned
	PermissionDenied   = BaseErrorWrapper(codes.PermissionDenied)   // PermissionDenied as name mentioned
	ResourceExhausted  = BaseErrorWrapper(codes.ResourceExhausted)  // ResourceExhausted as name mentioned
	FailedPrecondition = BaseErrorWrapper(codes.FailedPrecondition) // FailedPrecondition as name mentioned
	Aborted            = BaseErrorWrapper(codes.Aborted)            // Aborted as name mentioned
	OutOfRange         = BaseErrorWrapper(codes.OutOfRange)         // OutOfRange as name mentioned
	Unimplemented      = BaseErrorWrapper(codes.Unimplemented)      // Unimplemented as name mentioned
	Internal           = BaseErrorWrapper(codes.Internal)           // Internal as name mentioned
	Unavailable        = BaseErrorWrapper(codes.Unavailable)        // Unavailable as name mentioned
	DataLoss           = BaseErrorWrapper(codes.DataLoss)           // DataLoss as name mentioned
	Unauthenticated    = BaseErrorWrapper(codes.Unauthenticated)    // Unauthenticated as name mentioned
)

// BaseErrorWrapper will wrap grpc code into ErrorWrapper
func BaseErrorWrapper(code codes.Code) ErrorWrapper {
	return func(msg string, errors ...error) *status.Status {
		st := status.New(code, msg)

		// Inject grpc error as detail
		st, _ = st.WithDetails(&rk_error.ErrorDetail{
			Code:    int32(code),
			Status:  code.String(),
			Message: fmt.Sprintf("[from-grpc] %s", msg),
		})

		for i := range errors {
			st1, _ := status.FromError(errors[i])
			detail := &rk_error.ErrorDetail{
				Code:    int32(st1.Code()),
				Status:  st1.Code().String(),
				Message: st1.Message(),
			}
			st, _ = st.WithDetails(detail)
		}

		return st
	}
}
