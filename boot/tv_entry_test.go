// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.

package rkgrpc

import (
	"context"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/stretchr/testify/assert"
	httptest "github.com/stretchr/testify/http"
	"net/http"
	"net/url"
	"testing"
)

func TestNewTvEntry(t *testing.T) {
	entry := NewTvEntry(
		WithEventLoggerEntryTv(rkentry.NoopEventLoggerEntry()),
		WithZapLoggerEntryTv(rkentry.NoopZapLoggerEntry()))

	assert.Equal(t, TvEntryNameDefault, entry.GetName())
	assert.Equal(t, TvEntryType, entry.GetType())
	assert.Equal(t, TvEntryDescription, entry.GetDescription())
	assert.NotEmpty(t, entry.String())
	assert.Nil(t, entry.UnmarshalJSON(nil))
}

func TestTvEntry_Bootstrap(t *testing.T) {
	entry := NewTvEntry(
		WithEventLoggerEntryTv(rkentry.NoopEventLoggerEntry()),
		WithZapLoggerEntryTv(rkentry.NoopZapLoggerEntry()))

	entry.Bootstrap(context.TODO())
}

func TestTvEntry_Interrupt(t *testing.T) {
	entry := NewTvEntry(
		WithEventLoggerEntryTv(rkentry.NoopEventLoggerEntry()),
		WithZapLoggerEntryTv(rkentry.NoopZapLoggerEntry()))

	entry.Interrupt(context.TODO())
}

func TestTvEntry_TV(t *testing.T) {
	entry := NewTvEntry(
		WithEventLoggerEntryTv(rkentry.NoopEventLoggerEntry()),
		WithZapLoggerEntryTv(rkentry.NoopZapLoggerEntry()))
	entry.Bootstrap(context.TODO())

	defer assertNotPanic(t)
	// With all paths
	w := &httptest.TestResponseWriter{}
	r := &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/apis",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/entries",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/configs",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/certs",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/os",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/env",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/prometheus",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/logs",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/deps",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/license",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/info",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/git",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/gwErrorMapping",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)

	w = &httptest.TestResponseWriter{}
	r = &http.Request{
		URL: &url.URL{
			Path: "/rk/v1/tv/unknown",
		},
	}
	entry.TV(w, r)
	assert.NotEmpty(t, w.Output)
}
