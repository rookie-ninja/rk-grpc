// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpcerr

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"testing"
)

func TestBaseErrorWrapper(t *testing.T) {
	wrap := BaseErrorWrapper(codes.Canceled)
	status := wrap("ut message", errors.New("ut error"))
	assert.Equal(t, "ut message", status.Message())
}
