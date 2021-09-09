// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

// Package rkgrpc is for includes assets and api files into pkger
package rkgrpc

import "github.com/markbates/pkger"

func init() {
	pkger.Include("/boot/assets")
	pkger.Include("/boot/api")
}
