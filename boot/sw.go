// Copyright (c) 2020 rookie-ninja
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
package rk_grpc

import (
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const (
	swHandlerPrefix = "/swagger/"
	gwHandlerPrefix = "/"
	swAssetsPath    = "./assets/swagger-ui/"
)

var (
	swaggerIndexHTML = `<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>RK Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/3.35.1/swagger-ui.css" >
    <link rel="icon" type="image/png" href="https://editor.swagger.io/dist/favicon-32x32.png" sizes="32x32" />
    <link rel="icon" type="image/png" href="https://editor.swagger.io/dist/favicon-32x32.png" sizes="16x16" />
    <style>
      html
      {
        box-sizing: border-box;
        overflow: -moz-scrollbars-vertical;
        overflow-y: scroll;
      }

      *,
      *:before,
      *:after
      {
        box-sizing: inherit;
      }

      body
      {
        margin:0;
        background: #fafafa;
      }
    </style>
  </head>

  <body>
    <div id="swagger-ui"></div>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/3.35.1/swagger-ui-bundle.js"> </script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/3.35.1/swagger-ui-standalone-preset.js"> </script>
    <script>
    window.onload = function() {
      // Begin Swagger UI call region
      const ui = SwaggerUIBundle({
          configUrl: "swagger-config.json",
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [
              SwaggerUIBundle.presets.apis,
              SwaggerUIStandalonePreset
          ],
          plugins: [
              SwaggerUIBundle.plugins.DownloadUrl
          ],
          layout: "StandaloneLayout"
      })
      // End Swagger UI call region

      window.ui = ui
    }
  </script>
  </body>
</html>
`
	commonServiceJson = `{
  "swagger": "2.0",
  "info": {
    "title": "api/v1/rk_common_service.proto",
    "version": "version not set"
  },
  "consumes": [
    "application/json"
  ],
  "produces": [
    "application/json"
  ],
  "paths": {
    "/v1/rk/apis": {
      "get": {
        "summary": "List API Stub",
        "operationId": "RkCommonService_APIS",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1ListAPIResponse"
            }
          },
          "default": {
            "description": "An unexpected error response.",
            "schema": {
              "$ref": "#/definitions/rpcStatus"
            }
          }
        },
        "tags": [
          "RkCommonService"
        ]
      }
    },
    "/v1/rk/config": {
      "get": {
        "summary": "Config Stub",
        "operationId": "RkCommonService_Config",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1DumpConfigResponse"
            }
          },
          "default": {
            "description": "An unexpected error response.",
            "schema": {
              "$ref": "#/definitions/rpcStatus"
            }
          }
        },
        "tags": [
          "RkCommonService"
        ]
      }
    },
    "/v1/rk/gc": {
      "get": {
        "summary": "GC Stub",
        "operationId": "RkCommonService_GC",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1GCResponse"
            }
          },
          "default": {
            "description": "An unexpected error response.",
            "schema": {
              "$ref": "#/definitions/rpcStatus"
            }
          }
        },
        "tags": [
          "RkCommonService"
        ]
      }
    },
    "/v1/rk/healthy": {
      "get": {
        "summary": "Healthy Stub",
        "operationId": "RkCommonService_Healthy",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1HealthyResponse"
            }
          },
          "default": {
            "description": "An unexpected error response.",
            "schema": {
              "$ref": "#/definitions/rpcStatus"
            }
          }
        },
        "tags": [
          "RkCommonService"
        ]
      }
    },
    "/v1/rk/info": {
      "get": {
        "summary": "Info Stub",
        "operationId": "RkCommonService_Info",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1InfoResponse"
            }
          },
          "default": {
            "description": "An unexpected error response.",
            "schema": {
              "$ref": "#/definitions/rpcStatus"
            }
          }
        },
        "tags": [
          "RkCommonService"
        ]
      }
    },
    "/v1/rk/sys": {
      "get": {
        "summary": "Sys Stub",
        "operationId": "RkCommonService_Sys",
        "responses": {
          "200": {
            "description": "A successful response.",
            "schema": {
              "$ref": "#/definitions/v1SysResponse"
            }
          },
          "default": {
            "description": "An unexpected error response.",
            "schema": {
              "$ref": "#/definitions/rpcStatus"
            }
          }
        },
        "tags": [
          "RkCommonService"
        ]
      }
    }
  },
  "definitions": {
    "protobufAny": {
      "type": "object",
      "properties": {
        "type_url": {
          "type": "string"
        },
        "value": {
          "type": "string",
          "format": "byte"
        }
      }
    },
    "rpcStatus": {
      "type": "object",
      "properties": {
        "code": {
          "type": "integer",
          "format": "int32"
        },
        "message": {
          "type": "string"
        },
        "details": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/protobufAny"
          }
        }
      }
    },
    "v1API": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        },
        "grpc": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1GRPC"
          }
        },
        "gw": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1GW"
          }
        }
      }
    },
    "v1DumpConfigResponse": {
      "type": "object",
      "properties": {
        "viper": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1Viper"
          }
        },
        "rk": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1RK"
          }
        }
      }
    },
    "v1GCResponse": {
      "type": "object",
      "properties": {
        "mem_stats_before_gc": {
          "$ref": "#/definitions/v1MemStats"
        },
        "mem_stats_after_gc": {
          "$ref": "#/definitions/v1MemStats"
        }
      },
      "title": "GC response, memory stats would be returned"
    },
    "v1GRPC": {
      "type": "object",
      "properties": {
        "service": {
          "type": "string"
        },
        "method": {
          "type": "string"
        },
        "type": {
          "type": "string"
        },
        "port": {
          "type": "integer",
          "format": "int64"
        }
      }
    },
    "v1GW": {
      "type": "object",
      "properties": {
        "port": {
          "type": "integer",
          "format": "int64"
        },
        "path": {
          "type": "string"
        },
        "method": {
          "type": "string"
        },
        "sw": {
          "type": "string"
        }
      }
    },
    "v1HealthyResponse": {
      "type": "object",
      "properties": {
        "healthy": {
          "type": "boolean"
        }
      }
    },
    "v1Info": {
      "type": "object",
      "properties": {
        "uid": {
          "type": "string"
        },
        "gid": {
          "type": "string"
        },
        "username": {
          "type": "string"
        },
        "start_time": {
          "type": "string"
        },
        "up_time_sec": {
          "type": "integer",
          "format": "int64"
        },
        "up_time_str": {
          "type": "string"
        },
        "application": {
          "type": "string"
        },
        "realm": {
          "type": "string"
        },
        "region": {
          "type": "string"
        },
        "az": {
          "type": "string"
        },
        "domain": {
          "type": "string"
        }
      }
    },
    "v1InfoResponse": {
      "type": "object",
      "properties": {
        "info": {
          "$ref": "#/definitions/v1Info"
        }
      }
    },
    "v1ListAPIResponse": {
      "type": "object",
      "properties": {
        "api": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1API"
          }
        }
      }
    },
    "v1MemStats": {
      "type": "object",
      "properties": {
        "mem_alloc_byte": {
          "type": "integer",
          "format": "int64",
          "description": "Alloc is bytes of allocated heap objects."
        },
        "sys_alloc_byte": {
          "type": "integer",
          "format": "int64",
          "description": "Sys is the total bytes of memory obtained from the OS."
        },
        "mem_usage_percentage": {
          "type": "number",
          "format": "float",
          "title": "memory usage"
        },
        "last_gc_timestamp": {
          "type": "string",
          "title": "LastGC is the time the last garbage collection finished.\nRepresent as RFC3339 time format"
        },
        "gc_count_total": {
          "type": "integer",
          "format": "int64",
          "description": "The number of completed GC cycles."
        },
        "force_gc_count": {
          "type": "integer",
          "format": "int64",
          "description": "/ The number of GC cycles that were forced by\nthe application calling the GC function."
        }
      },
      "title": "Memory stats"
    },
    "v1RK": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        },
        "raw": {
          "type": "string"
        }
      }
    },
    "v1ReqMetricsRK": {
      "type": "object",
      "properties": {
        "path": {
          "type": "string"
        },
        "elapsed_nano_p50": {
          "type": "number",
          "format": "float"
        },
        "elapsed_nano_p90": {
          "type": "number",
          "format": "float"
        },
        "elapsed_nano_p99": {
          "type": "number",
          "format": "float"
        },
        "elapsed_nano_p999": {
          "type": "number",
          "format": "float"
        },
        "count": {
          "type": "integer",
          "format": "int64"
        },
        "res_code": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1ResCodeRK"
          }
        }
      }
    },
    "v1ReqResponse": {
      "type": "object",
      "properties": {
        "metrics": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/v1ReqMetricsRK"
          }
        }
      }
    },
    "v1ResCodeRK": {
      "type": "object",
      "properties": {
        "res_code": {
          "type": "string"
        },
        "count": {
          "type": "integer",
          "format": "int64"
        }
      }
    },
    "v1SysResponse": {
      "type": "object",
      "properties": {
        "cpu_percentage": {
          "type": "number",
          "format": "float"
        },
        "mem_percentage": {
          "type": "number",
          "format": "float"
        },
        "mem_usage_mb": {
          "type": "integer",
          "format": "int64"
        },
        "up_time": {
          "type": "string"
        }
      }
    },
    "v1Viper": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string"
        },
        "raw": {
          "type": "string"
        }
      }
    }
  }
}
`
	swaggerConfigJson = ``
	swaggerJsonFiles  = make(map[string]string, 0)
)

type swURLConfig struct {
	URLs []*swURL `json:"urls"`
}

type swURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type swEntry struct {
	sourceType string
	logger     *zap.Logger
	jsonPath   string
	path       string
	headers    map[string]string
}

type swOption func(*swEntry)

func withSWPath(path string) swOption {
	return func(entry *swEntry) {
		if len(path) < 1 {
			path = "sw"
		}
		entry.path = path
	}
}

func withSWJsonPath(path string) swOption {
	return func(entry *swEntry) {
		entry.jsonPath = path
	}
}

func withHeaders(headers map[string]string) swOption {
	return func(entry *swEntry) {
		entry.headers = headers
	}
}

func newSWEntry(opts ...swOption) *swEntry {
	entry := &swEntry{
		logger: zap.NewNop(),
	}

	for i := range opts {
		opts[i](entry)
	}

	// Deal with Path
	// add "/" at start and end side if missing
	if !strings.HasPrefix(entry.path, "/") {
		entry.path = "/" + entry.path
	}

	if !strings.HasSuffix(entry.path, "/") {
		entry.path = entry.path + "/"
	}

	// init swagger configs
	entry.initSwaggerConfig()

	return entry
}

func (entry *swEntry) GetPath() string {
	return entry.path
}

func (entry *swEntry) Start() {
	// Init http server
	gwMux := runtime.NewServeMux()

	// Init swagger http mux
	httpMux := http.NewServeMux()
	httpMux.Handle(gwHandlerPrefix, gwMux)
	httpMux.HandleFunc(swHandlerPrefix, entry.swJsonFileHandler)
	httpMux.HandleFunc(entry.path, entry.swIndexHandler)
}

func (entry *swEntry) initSwaggerConfig() {
	// 1: Get swagger-config.json if exists
	swaggerURLConfig := &swURLConfig{
		URLs: make([]*swURL, 0),
	}

	// 2: Add user API swagger JSON
	entry.listFilesWithSuffix()
	for k, _ := range swaggerJsonFiles {
		swaggerURL := &swURL{
			Name: k,
			URL:  path.Join("/swagger", k),
		}
		entry.appendAndDeduplication(swaggerURLConfig, swaggerURL)
	}

	// 3: Add pl-common
	entry.appendAndDeduplication(swaggerURLConfig, &swURL{
		Name: "rk-common",
		URL:  "/swagger/rk_common_service.swagger.json",
	})

	// 4: Marshal to swagger-config.json
	bytes, err := json.Marshal(swaggerURLConfig)
	if err != nil {
		entry.logger.Warn("failed to unmarshal swagger-config.json",
			zap.String("sw_path", entry.path),
			zap.String("sw_assets_path", swAssetsPath),
			zap.Error(err))
		shutdownWithError(err)
	}

	swaggerConfigJson = string(bytes)
}

func (entry *swEntry) listFilesWithSuffix() {
	jsonPath := entry.jsonPath
	suffix := ".json"
	// re-path it with working directory if not absolute path
	if !path.IsAbs(entry.jsonPath) {
		wd, err := os.Getwd()
		if err != nil {
			entry.logger.Info("failed to get working directory",
				zap.String("error", err.Error()))
			shutdownWithError(err)
		}
		jsonPath = path.Join(wd, jsonPath)
	}

	files, err := ioutil.ReadDir(jsonPath)
	if err != nil {
		entry.logger.Error("failed to list files with suffix",
			zap.String("path", jsonPath),
			zap.String("suffix", suffix),
			zap.String("error", err.Error()))
		shutdownWithError(err)
	}

	for i := range files {
		file := files[i]
		if !file.IsDir() && strings.HasSuffix(file.Name(), suffix) {
			bytes, err := ioutil.ReadFile(path.Join(jsonPath, file.Name()))
			if err != nil {
				entry.logger.Info("failed to read file with suffix",
					zap.String("path", path.Join(jsonPath, file.Name())),
					zap.String("suffix", suffix),
					zap.String("error", err.Error()))
				shutdownWithError(err)
			}

			swaggerJsonFiles[file.Name()] = string(bytes)
		}
	}
}

func (entry *swEntry) appendAndDeduplication(config *swURLConfig, url *swURL) {
	urls := config.URLs

	for i := range urls {
		element := urls[i]

		if element.Name == url.Name {
			return
		}
	}

	config.URLs = append(config.URLs, url)
}

func (entry *swEntry) swJsonFileHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "swagger.json") {
		http.NotFound(w, r)
		return
	}

	p := strings.TrimPrefix(r.URL.Path, swHandlerPrefix)

	// This is common file
	if p == "rk_common_service.swagger.json" {
		http.ServeContent(w, r, "rk-common", time.Now(), strings.NewReader(commonServiceJson))
		return
	}

	// Set no-cache headers by default
	w.Header().Set("cache-control", "no-cache")

	for k, v := range entry.headers {
		w.Header().Set(k, v)
	}

	value, ok := swaggerJsonFiles[p]

	if ok {
		http.ServeContent(w, r, p, time.Now(), strings.NewReader(value))
	}
}

func (entry *swEntry) swIndexHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), "/")
	// This is common file
	if path == "sw" {
		http.ServeContent(w, r, "index.html", time.Now(), strings.NewReader(swaggerIndexHTML))
		return
	} else if path == "sw/swagger-config.json" {
		http.ServeContent(w, r, "swagger-config.json", time.Now(), strings.NewReader(swaggerConfigJson))
		return
	} else {

	}
}
