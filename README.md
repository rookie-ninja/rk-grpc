# rk-grpc
[![build](https://github.com/rookie-ninja/rk-grpc/actions/workflows/ci.yml/badge.svg)](https://github.com/rookie-ninja/rk-grpc/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/rookie-ninja/rk-grpc/branch/master/graph/badge.svg?token=08TCFIIVS0)](https://codecov.io/gh/rookie-ninja/rk-grpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/rookie-ninja/rk-grpc)](https://goreportcard.com/report/github.com/rookie-ninja/rk-grpc)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Middlewares & bootstrapper designed for [gRPC](https://grpc.io/docs/languages/go/) and [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway). [Documentation](https://rkdev.info/docs/bootstrapper/user-guide/grpc-golang/).

This belongs to [rk-boot](https://github.com/rookie-ninja/rk-boot) family. We suggest use this lib from [rk-boot](https://github.com/rookie-ninja/rk-boot).

![image](docs/img/boot-arch.png)

## Architecture
![image](docs/img/grpc-arch.png)

## Supported bootstrap
| Bootstrap  | Description                                                                                                                                |
|------------|--------------------------------------------------------------------------------------------------------------------------------------------|
| YAML based | Start [gRPC](https://grpc.io/docs/languages/go/) and [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) microservice from YAML |
| Code based | Start [gRPC](https://grpc.io/docs/languages/go/) and [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) microservice from code |

## Supported instances
All instances could be configured via YAML or Code.

**User can enable anyone of those as needed! No mandatory binding!**

| Instance                                                               | Description                                                                                                                    |
|------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------|
| [gRPC](https://grpc.io/docs/languages/go/)                             | [gRPC](https://grpc.io/docs/languages/go/) defined with protocol buffer.                                                       |
| [gRPC](https://grpc.io/docs/languages/go/) proxy                       | Proxy gRPC request to another gRPC server.                                                                                     |
| [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)         | [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) service with same port.                                         |
| [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) options | Well defined [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) options.                                           |
| Config                                                                 | Configure [spf13/viper](https://github.com/spf13/viper) as config instance and reference it from YAML                          |
| Logger                                                                 | Configure [uber-go/zap](https://github.com/uber-go/zap) logger configuration and reference it from YAML                        |
| Event                                                                  | Configure logging of RPC with [rk-query](https://github.com/rookie-ninja/rk-query) and reference it from YAML                  |
| Cert                                                                   | Fetch TLS/SSL certificates from remote datastore like ETCD and start microservice.                                             |
| Prometheus                                                             | Start prometheus client at client side and push metrics to [pushgateway](https://github.com/prometheus/pushgateway) as needed. |
| Swagger                                                                | Builtin swagger UI handler.                                                                                                    |
| Docs                                                                   | Builtin [RapiDoc](https://github.com/mrin9/RapiDoc) instance which can be used to replace swagger and RK TV.                   |
| CommonService                                                          | List of common APIs.                                                                                                           |
| StaticFileHandler                                                      | A Web UI shows files could be downloaded from server, currently support source of local and embed.FS.                          |
| PProf                                                                  | PProf web UI.                                                                                                                  |

## Supported middlewares
All middlewares could be configured via YAML or Code.

**User can enable anyone of those as needed! No mandatory binding!**

| Middleware | Description                                                                                                                                           |
|------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| Metrics    | Collect RPC metrics and export to [prometheus](https://github.com/prometheus/client_golang) client.                                                   |
| Log        | Log every RPC requests as event with [rk-query](https://github.com/rookie-ninja/rk-query).                                                            |
| Trace      | Collect RPC trace and export it to stdout, file or jaeger with [open-telemetry/opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go). |
| Panic      | Recover from panic for RPC requests and log it.                                                                                                       |
| Meta       | Send micsroservice metadata as header to client.                                                                                                      |
| Auth       | Support [Basic Auth] and [API Key] authorization types.                                                                                               |
| RateLimit  | Limiting RPC rate globally or per path.                                                                                                               |
| Timeout    | Timing out request by configuration.                                                                                                                  |
| CORS       | Server side CORS validation.                                                                                                                          |
| JWT        | Server side JWT validation.                                                                                                                           |
| Secure     | Server side secure validation.                                                                                                                        |
| CSRF       | Server side csrf validation.                                                                                                                          |

## Installation
`go get github.com/rookie-ninja/rk-grpc/v2`

## Quick Start
In the bellow example, we will start microservice with bellow functionality and middlewares enabled via YAML.

- [gRPC](https://grpc.io/docs/languages/go/) and [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) server
- [gRPC](https://grpc.io/docs/languages/go/) server reflection
- Swagger UI
- CommonService
- Docs
- Prometheus Metrics (middleware)
- Logging (middleware)
- Meta (middleware)

Please refer example at [example/boot/simple](example/boot/simple).

### 1.Prepare .proto files
- api/v1/greeter.proto

```protobuf
syntax = "proto3";

package api.v1;

option go_package = "api/v1/greeter";

service Greeter {
  rpc Greeter (GreeterRequest) returns (GreeterResponse) {}
}

message GreeterRequest {
  bytes msg = 1;
}

message GreeterResponse {}
```

- api/v1/gw_mapping.yaml

```yaml
type: google.api.Service
config_version: 3

# Please refer google.api.Http in https://github.com/googleapis/googleapis/blob/master/google/api/http.proto file for details.
http:
  rules:
    - selector: api.v1.Greeter.Greeter
      get: /v1/greeter
```

- buf.yaml

```yaml
version: v1beta1
name: github.com/rk-dev/rk-boot
build:
  roots:
    - api
```

- buf.gen.yaml

```yaml
version: v1beta1
plugins:
  # protoc-gen-go needs to be installed, generate go files based on proto files
  - name: go
    out: api/gen
    opt:
     - paths=source_relative
  # protoc-gen-go-grpc needs to be installed, generate grpc go files based on proto files
  - name: go-grpc
    out: api/gen
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false
  # protoc-gen-grpc-gateway needs to be installed, generate grpc-gateway go files based on proto files
  - name: grpc-gateway
    out: api/gen
    opt:
      - paths=source_relative
      - grpc_api_configuration=api/v1/gw_mapping.yaml
  # protoc-gen-openapiv2 needs to be installed, generate swagger config files based on proto files
  - name: openapiv2
    out: api/gen
    opt:
      - grpc_api_configuration=api/v1/gw_mapping.yaml
```

### 2.Generate .pb.go files with [buf](https://docs.buf.build/introduction)
```
$ buf generate --path api/v1
```

- directory hierarchy

```
.
├── api
│   ├── gen
│   │   └── v1
│   │       ├── greeter.pb.go
│   │       ├── greeter.pb.gw.go
│   │       ├── greeter.swagger.json
│   │       └── greeter_grpc.pb.go
│   └── v1
│       ├── greeter.proto
│       └── gw_mapping.yaml
├── boot.yaml
├── buf.gen.yaml
├── buf.yaml
├── go.mod
├── go.sum
└── main.go
```

### 3.Create boot.yaml
```yaml
---
grpc:
  - name: greeter                     # Required
    port: 8080                        # Required
    enabled: true                     # Required
    enableReflection: true            # Optional, default: false
    enableRkGwOption: true            # Optional, default: false
    commonService:
      enabled: true                   # Optional, default: false
    docs:
      enabled: true                   # Optional, default: false
    sw:
      enabled: true                   # Optional, default: false
    prom:
      enabled: true                   # Optional, default: false
    middleware:
      logging:
        enabled: true                 # Optional, default: false
      prom:
        enabled: true                 # Optional, default: false
      meta:
        enabled: true                 # Optional, default: false
```

### 4.Create main.go

```go
// Copyright (c) 2021 rookie-ninja
//
// Use of this source code is governed by an Apache-style
// license that can be found in the LICENSE file.
package main

import (
  "context"
  "embed"
  _ "embed"
  "github.com/rookie-ninja/rk-entry/v2/entry"
  "github.com/rookie-ninja/rk-grpc/v2/boot"
  proto "github.com/rookie-ninja/rk-grpc/v2/example/boot/simple/api/gen/v1"
  "google.golang.org/grpc"
)

//go:embed boot.yaml
var boot []byte

//go:embed api/gen/v1
var docsFS embed.FS

//go:embed api/gen/v1
var staticFS embed.FS

func init() {
  rkentry.GlobalAppCtx.AddEmbedFS(rkentry.DocsEntryType, "greeter", &docsFS)
  rkentry.GlobalAppCtx.AddEmbedFS(rkentry.SWEntryType, "greeter", &docsFS)
  rkentry.GlobalAppCtx.AddEmbedFS(rkentry.StaticFileHandlerEntryType, "greeter", &staticFS)
}

func main() {
  // Bootstrap basic entries from boot config.
  rkentry.BootstrapPreloadEntryYAML(boot)

  // Bootstrap grpc entry from boot config
  res := rkgrpc.RegisterGrpcEntryYAML(boot)

  // Get GrpcEntry
  grpcEntry := res["greeter"].(*rkgrpc.GrpcEntry)
  // Register gRPC server
  grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
    proto.RegisterGreeterServer(server, &GreeterServer{})
  })
  // Register grpc-gateway func
  grpcEntry.AddRegFuncGw(proto.RegisterGreeterHandlerFromEndpoint)

  // Bootstrap grpc entry
  grpcEntry.Bootstrap(context.Background())

  // Wait for shutdown signal
  rkentry.GlobalAppCtx.WaitForShutdownSig()

  // Interrupt gin entry
  grpcEntry.Interrupt(context.Background())
}

// GreeterServer Implementation of GreeterServer.
type GreeterServer struct{}

// Greeter Handle Greeter method.
func (server *GreeterServer) Greeter(context.Context, *proto.GreeterRequest) (*proto.GreeterResponse, error) {
  return &proto.GreeterResponse{}, nil
}
```

### 5.Start server
```
$ go run main.go
```

### 6.Validation
#### 6.1 gRPC & grpc-gateway server
Try to test [gRPC](https://grpc.io/docs/languages/go/) & [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) Service with [curl](https://curl.se/) & [grpcurl](https://github.com/fullstorydev/grpcurl)

```shell script
# Curl to common service
$ curl localhost:8080/rk/v1/ready
{"ready":true}
```

#### 6.2 Swagger UI
Please refer **sw** section at [Full YAML](#full-yaml).

By default, we could access swagger UI at [http://localhost:8080/sw](http://localhost:8080/sw)

![sw](docs/img/simple-sw.png)

#### 4.3 Docs UI
Please refer **docs** section at [Full YAML](#full-yaml).

By default, we could access docs UI at [http://localhost:8080/docs](http://localhost:8080/docs)

![docs](docs/img/simple-docs.png)

#### 6.4 Prometheus Metrics
Please refer **middleware.prom** section at [Full YAML](#full-yaml).

By default, we could access prometheus client at [http://localhost:8080/metrics](http://localhost:8080/metrics)

![prom](docs/img/simple-prom.png)

#### 6.5 Logging
Please refer **middleware.logging** section at [Full YAML](#full-yaml).

By default, we enable zap logger and event logger with encoding type of [console]. Encoding type of [json] and [flatten] is also supported.

```shell script
2021-12-28T05:36:21.561+0800    INFO    boot/grpc_entry.go:1515 Bootstrap grpcEntry     {"eventId": "db2c977c-e0ff-4b21-bc0d-5966f1cad093", "entryName": "greeter"}
------------------------------------------------------------------------
endTime=2021-12-28T05:36:21.563575+08:00
startTime=2021-12-28T05:36:21.561362+08:00
elapsedNano=2213846
timezone=CST
ids={"eventId":"db2c977c-e0ff-4b21-bc0d-5966f1cad093"}
app={"appName":"rk","appVersion":"","entryName":"greeter","entryType":"GrpcEntry"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"commonServiceEnabled":true,"commonServicePathPrefix":"/rk/v1/","grpcPort":8080,"gwPort":8080,"promEnabled":true,"promPath":"/metrics","promPort":8080,"swEnabled":true,"swPath":"/sw/","tvEnabled":true,"tvPath":"/rk/v1/tv/"}
error={}
counters={}
pairs={}
timing={}
remoteAddr=localhost
operation=Bootstrap
resCode=OK
eventStatus=Ended
EOE
```

#### 6.6 Meta
Please refer **meta** section at [Full YAML](#full-yaml).

By default, we will send back some metadata to client with headers.

```shell script
$ curl -vs localhost:8080/rk/v1/ready
...
< HTTP/1.1 200 OK
< Content-Type: application/json
< X-Request-Id: 7e4f5ac5-3369-485f-89f7-55551cc4a9a1
< X-Rk-App-Name: rk
< X-Rk-App-Unix-Time: 2021-12-28T05:39:50.508328+08:00
< X-Rk-App-Version: 
< X-Rk-Received-Time: 2021-12-28T05:39:50.508328+08:00
< Date: Mon, 27 Dec 2021 21:39:50 GMT
...
```

#### 6.7 Send request
We registered /v1/greeter API in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) server and let's validate it!

```shell script
$ curl -vs localhost:8080/v1/greeter             
*   Trying ::1...
* TCP_NODELAY set
* Connection failed
* connect to ::1 port 8080 failed: Connection refused
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8080 (#0)
> GET /v1/greeter HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.64.1
> Accept: */*
> 
< HTTP/1.1 200 OK
< Content-Type: application/json
< X-Request-Id: 07b0fbf6-cebf-40ac-84a2-533bbd4b8958
< X-Rk-App-Name: rk
< X-Rk-App-Unix-Time: 2021-12-28T05:41:04.653652+08:00
< X-Rk-App-Version: 
< X-Rk-Received-Time: 2021-12-28T05:41:04.653652+08:00
< Date: Mon, 27 Dec 2021 21:41:04 GMT
< Content-Length: 2
< 
* Connection #0 to host localhost left intact
{}
```

We registered api.v1.Greeter.Greeter API in [gRPC](https://grpc.io/docs/languages/go/) server and let's validate it!

```shell script
$ grpcurl -plaintext localhost:8080 api.v1.Greeter.Greeter 
{}
```

#### 6.8 RPC logs
Bellow logs would be printed in stdout.

The first block of log is from [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) request.

The second block of log is from [gRPC](https://grpc.io/docs/languages/go/) request.

```
------------------------------------------------------------------------
endTime=2021-12-28T05:45:52.986041+08:00
startTime=2021-12-28T05:45:52.985956+08:00
elapsedNano=85065
timezone=CST
ids={"eventId":"88362f69-7eda-4f03-bdbe-7ef667d06bac","requestId":"88362f69-7eda-4f03-bdbe-7ef667d06bac"}
app={"appName":"rk","appVersion":"","entryName":"greeter","entryType":"GrpcEntry"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"Greeter","grpcService":"api.v1.Greeter","grpcType":"unaryServer","gwMethod":"GET","gwPath":"/v1/greeter","gwScheme":"http","gwUserAgent":"curl/7.64.1"}
error={}
counters={}
pairs={}
timing={}
remoteAddr=127.0.0.1:61520
operation=/api.v1.Greeter/Greeter
resCode=OK
eventStatus=Ended
EOE
------------------------------------------------------------------------
endTime=2021-12-28T05:44:45.686734+08:00
startTime=2021-12-28T05:44:45.686592+08:00
elapsedNano=141716
timezone=CST
ids={"eventId":"7765862c-9e83-443a-a6e5-bb28f17f8ea0","requestId":"7765862c-9e83-443a-a6e5-bb28f17f8ea0"}
app={"appName":"rk","appVersion":"","entryName":"greeter","entryType":"GrpcEntry"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"Greeter","grpcService":"api.v1.Greeter","grpcType":"unaryServer","gwMethod":"","gwPath":"","gwScheme":"","gwUserAgent":""}
error={}
counters={}
pairs={}
timing={}
remoteAddr=127.0.0.1:57149
operation=/api.v1.Greeter/Greeter
resCode=OK
eventStatus=Ended
EOE
```

#### 6.9 RPC prometheus metrics
Prometheus client will automatically register into [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) instance at /metrics.

Access [http://localhost:8080/metrics](http://localhost:8080/metrics)

![image](docs/img/prom-inter.png)

## YAML options
User can start multiple [gRPC](https://grpc.io/docs/languages/go/) and [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) instances at the same time. Please make sure use different port and name.

### gRPC Service
| name                    | description                                                                                                                                  | type    | default value           |
|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|---------|-------------------------|
| grpc.name               | Required, The name of [gRPC](https://grpc.io/docs/languages/go/) server                                                                      | string  | N/A                     |
| grpc.enabled            | Required, Enable [gRPC](https://grpc.io/docs/languages/go/) entry                                                                            | bool    | false                   |
| grpc.port               | Required, The port of [gRPC](https://grpc.io/docs/languages/go/) server                                                                      | integer | nil, server won't start |
| grpc.description        | Optional, Description of [gRPC](https://grpc.io/docs/languages/go/) entry.                                                                   | string  | ""                      |
| grpc.enableReflection   | Optional, Enable [gRPC](https://grpc.io/docs/languages/go/) server reflection                                                                | boolean | false                   |
| grpc.enableRkGwOption   | Optional, Enable RK style [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) server options. [detail](boot/gw_server_options.go) | false   |
| grpc.noRecvMsgSizeLimit | Optional, Disable [gRPC](https://grpc.io/docs/languages/go/) server side receive message size limit                                          | false   |
| grpc.certEntry          | Optional, Reference of certEntry declared in [cert entry](https://github.com/rookie-ninja/rk-entry#certentry)                                | string  | ""                      |
| grpc.loggerEntry        | Optional, Reference of loggerEntry declared in [LoggerEntry](https://github.com/rookie-ninja/rk-entry#loggerentry)                           | string  | ""                      |
| grpc.eventEntry         | Optional, Reference of eventLEntry declared in [eventEntry](https://github.com/rookie-ninja/rk-entry#evententry)                             | string  | ""                      |

### gRPC gateway options
Please refer to bellow repository for detailed explanations.
- [protobuf-go/encoding/protojson/encode.go](https://github.com/protocolbuffers/protobuf-go/blob/master/encoding/protojson/encode.go#L43)
- [protobuf-go/encoding/protojson/decode.go ](https://github.com/protocolbuffers/protobuf-go/blob/master/encoding/protojson/decode.go#L33)

| name                                   | description                                                                                                   | type   | default value |
|----------------------------------------|---------------------------------------------------------------------------------------------------------------|--------|---------------|
| grpc.gwOption.marshal.multiline        | Optional, Enable multiline in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) marshaller       | bool   | false         |
| grpc.gwOption.marshal.emitUnpopulated  | Optional, Enable emitUnpopulated in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) marshaller | bool   | false         |
| grpc.gwOption.marshal.indent           | Optional, Set indent in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) marshaller             | string | "  "          |
| grpc.gwOption.marshal.allowPartial     | Optional, Enable allowPartial in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) marshaller    | bool   | false         |
| grpc.gwOption.marshal.useProtoNames    | Optional, Enable useProtoNames in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) marshaller   | bool   | false         |
| grpc.gwOption.marshal.useEnumNumbers   | Optional, Enable useEnumNumbers in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) marshaller  | bool   | false         |
| grpc.gwOption.unmarshal.allowPartial   | Optional, Enable allowPartial in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) unmarshaler   | bool   | false         |
| grpc.gwOption.unmarshal.discardUnknown | Optional, Enable discardUnknown in [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) unmarshaler | bool   | false         |

### Common Service
| Path         | Description                       |
|--------------|-----------------------------------|
| /rk/v1/gc    | Trigger GC                        |
| /rk/v1/ready | Get application readiness status. |
| /rk/v1/alive | Get application aliveness status. |
| /rk/v1/info  | Get application and process info. |

| name                         | description                             | type    | default value |
|------------------------------|-----------------------------------------|---------|---------------|
| gin.commonService.enabled    | Optional, Enable builtin common service | boolean | false         |
| gin.commonService.pathPrefix | Optional, Provide path prefix           | string  | /rk/v1        |

### Swagger
| name             | description                                                        | type     | default value |
|------------------|--------------------------------------------------------------------|----------|---------------|
| grpc.sw.enabled  | Optional, Enable swagger service over gin server                   | boolean  | false         |
| grpc.sw.path     | Optional, The path access swagger service from web                 | string   | /sw           |
| grpc.sw.jsonPath | Optional, Where the swagger.json files are stored locally          | string   | ""            |
| grpc.sw.headers  | Optional, Headers would be sent to caller as scheme of [key:value] | []string | []            |

### Docs (RapiDoc)
| name                  | description                                                                            | type     | default value |
|-----------------------|----------------------------------------------------------------------------------------|----------|---------------|
| grpc.docs.enabled     | Optional, Enable RapiDoc service over gin server                                       | boolean  | false         |
| grpc.docs.path        | Optional, The path access docs service from web                                        | string   | /docs         |
| grpc.docs.jsonPath    | Optional, Where the swagger.json or open API files are stored locally                  | string   | ""            |
| grpc.docs.headers     | Optional, Headers would be sent to caller as scheme of [key:value]                     | []string | []            |
| grpc.docs.style.theme | Optional, light and dark are supported options                                         | string   | []            |
| grpc.docs.debug       | Optional, Enable debugging mode in RapiDoc which can be used as the same as Swagger UI | boolean  | false         |

### Prom Client
| name                           | description                                                                        | type    | default value |
|--------------------------------|------------------------------------------------------------------------------------|---------|---------------|
| grpc.prom.enabled              | Optional, Enable prometheus                                                        | boolean | false         |
| grpc.prom.path                 | Optional, Path of prometheus                                                       | string  | /metrics      |
| grpc.prom.pusher.enabled       | Optional, Enable prometheus pusher                                                 | bool    | false         |
| grpc.prom.pusher.jobName       | Optional, Job name would be attached as label while pushing to remote pushgateway  | string  | ""            |
| grpc.prom.pusher.remoteAddress | Optional, PushGateWay address, could be form of http://x.x.x.x or x.x.x.x          | string  | ""            |
| grpc.prom.pusher.intervalMs    | Optional, Push interval in milliseconds                                            | string  | 1000          |
| grpc.prom.pusher.basicAuth     | Optional, Basic auth used to interact with remote pushgateway, form of [user:pass] | string  | ""            |
| grpc.prom.pusher.certEntry     | Optional, Reference of rkentry.CertEntry                                           | string  | ""            |

### Static file handler Service
| name                   | description                             | type    | default value |
|------------------------|-----------------------------------------|---------|---------------|
| grpc.static.enabled    | Optional, Enable static file handler    | boolean | false         |
| grpc.static.path       | Optional, path of static file handler   | string  | /static       |
| grpc.static.sourceType | Required, local and pkger supported     | string  | ""            |
| grpc.static.sourcePath | Required, full path of source directory | string  | ""            |

- About embed.FS
  User has to set embedFS before Bootstrap() function as bellow:
-
```go
//go:embed /*
var staticFS embed.FS

rkentry.GlobalAppCtx.AddEmbedFS(rkentry.StaticFileHandlerEntryType, "greeter", &staticFS)
```

### Middlewares
| name                   | description                                            | type     | default value |
|------------------------|--------------------------------------------------------|----------|---------------|
| grpc.middleware.ignore | The paths of prefix that will be ignored by middleware | []string | []            |

#### Logging
| name                                      | description                                            | type     | default value |
|-------------------------------------------|--------------------------------------------------------|----------|---------------|
| grpc.middleware.logging.enabled           | Enable log middleware                                  | boolean  | false         |
| grpc.middleware.logging.ignore            | The paths of prefix that will be ignored by middleware | []string | []            |
| grpc.middleware.logging.loggerEncoding    | json or console or flatten                             | string   | console       |
| grpc.middleware.logging.loggerOutputPaths | Output paths                                           | []string | stdout        |
| grpc.middleware.logging.eventEncoding     | json or console or flatten                             | string   | console       |
| grpc.middleware.logging.eventOutputPaths  | Output paths                                           | []string | false         |

We will log two types of log for every RPC call.
- Logger

Contains user printed logging with requestId or traceId.

- Event

Contains per RPC metadata, response information, environment information and etc.

| Field       | Description                                                                                                                                                                                                                                                                                                           |
|-------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| endTime     | As name described                                                                                                                                                                                                                                                                                                     |
| startTime   | As name described                                                                                                                                                                                                                                                                                                     |
| elapsedNano | Elapsed time for RPC in nanoseconds                                                                                                                                                                                                                                                                                   |
| timezone    | As name described                                                                                                                                                                                                                                                                                                     |
| ids         | Contains three different ids(eventId, requestId and traceId). If meta interceptor was enabled or event.SetRequestId() was called by user, then requestId would be attached. eventId would be the same as requestId if meta interceptor was enabled. If trace interceptor was enabled, then traceId would be attached. |
| app         | Contains [appName, appVersion](https://github.com/rookie-ninja/rk-entry#appinfoentry), entryName, entryType.                                                                                                                                                                                                          |
| env         | Contains arch, az, domain, hostname, localIP, os, realm, region. realm, region, az, domain were retrieved from environment variable named as REALM, REGION, AZ and DOMAIN. "*" means empty environment variable.                                                                                                      |
| payloads    | Contains RPC related metadata                                                                                                                                                                                                                                                                                         |
| error       | Contains errors if occur                                                                                                                                                                                                                                                                                              |
| counters    | Set by calling event.SetCounter() by user.                                                                                                                                                                                                                                                                            |
| pairs       | Set by calling event.AddPair() by user.                                                                                                                                                                                                                                                                               |
| timing      | Set by calling event.StartTimer() and event.EndTimer() by user.                                                                                                                                                                                                                                                       |
| remoteAddr  | As name described                                                                                                                                                                                                                                                                                                     |
| operation   | RPC method name                                                                                                                                                                                                                                                                                                       |
| resCode     | Response code of RPC                                                                                                                                                                                                                                                                                                  |
| eventStatus | Ended or InProgress                                                                                                                                                                                                                                                                                                   |

- example

```shell script
------------------------------------------------------------------------
endTime=2021-06-24T05:58:48.282193+08:00
startTime=2021-06-24T05:58:48.28204+08:00
elapsedNano=153005
timezone=CST
ids={"eventId":"573ce6a8-308b-4fc0-9255-33608b9e41d4","requestId":"573ce6a8-308b-4fc0-9255-33608b9e41d4"}
app={"appName":"rk-grpc","appVersion":"master-xxx","entryName":"greeter","entryType":"GrpcEntry"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.6","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"Healthy","grpcService":"rk.api.v1.RkCommonService","grpcType":"unaryServer","gwMethod":"GET","gwPath":"/rk/v1/healthy","gwScheme":"http","gwUserAgent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.77 Safari/537.36"}
error={}
counters={}
pairs={"healthy":"true"}
timing={}
remoteAddr=localhost:57135
operation=/rk.api.v1.RkCommonService/Healthy
resCode=OK
eventStatus=Ended
EOE
```

#### Prometheus
| name                         | description                                            | type     | default value |
|------------------------------|--------------------------------------------------------|----------|---------------|
| grpc.middleware.prom.enabled | Enable metrics middleware                              | boolean  | false         |
| grpc.middleware.prom.ignore  | The paths of prefix that will be ignored by middleware | []string | []            |

#### Auth
Enable the server side auth. codes.Unauthenticated would be returned to client if not authorized with user defined credential.

| name                         | description                                            | type     | default value |
|------------------------------|--------------------------------------------------------|----------|---------------|
| grpc.middleware.auth.enabled | Enable auth middleware                                 | boolean  | false         |
| grpc.middleware.auth.ignore  | The paths of prefix that will be ignored by middleware | []string | []            |
| grpc.middleware.auth.basic   | Basic auth credentials as scheme of <user:pass>        | []string | []            |
| grpc.middleware.auth.apiKey  | API key auth                                           | []string | []            |

#### Meta
Send application metadata as header to client.

| name                         | description                                            | type     | default value |
|------------------------------|--------------------------------------------------------|----------|---------------|
| grpc.middleware.meta.enabled | Enable meta middleware                                 | boolean  | false         |
| grpc.middleware.meta.ignore  | The paths of prefix that will be ignored by middleware | []string | []            |
| grpc.middleware.meta.prefix  | Header key was formed as X-<Prefix>-XXX                | string   | RK            |

#### Trace
| name                                                     | description                                            | type     | default value                    |
|----------------------------------------------------------|--------------------------------------------------------|----------|----------------------------------|
| grpc.middleware.trace.enabled                            | Enable tracing middleware                              | boolean  | false                            |
| grpc.middleware.trace.ignore                             | The paths of prefix that will be ignored by middleware | []string | []                               |
| grpc.middleware.trace.exporter.file.enabled              | Enable file exporter                                   | boolean  | false                            |
| grpc.middleware.trace.exporter.file.outputPath           | Export tracing info to files                           | string   | stdout                           |
| grpc.middleware.trace.exporter.jaeger.agent.enabled      | Export tracing info to jaeger agent                    | boolean  | false                            |
| grpc.middleware.trace.exporter.jaeger.agent.host         | As name described                                      | string   | localhost                        |
| grpc.middleware.trace.exporter.jaeger.agent.port         | As name described                                      | int      | 6831                             |
| grpc.middleware.trace.exporter.jaeger.collector.enabled  | Export tracing info to jaeger collector                | boolean  | false                            |
| grpc.middleware.trace.exporter.jaeger.collector.endpoint | As name described                                      | string   | http://localhost:16368/api/trace |
| grpc.middleware.trace.exporter.jaeger.collector.username | As name described                                      | string   | ""                               |
| grpc.middleware.trace.exporter.jaeger.collector.password | As name described                                      | string   | ""                               |

#### RateLimit
| name                                      | description                                                          | type     | default value |
|-------------------------------------------|----------------------------------------------------------------------|----------|---------------|
| grpc.middleware.rateLimit.enabled         | Enable rate limit middleware                                         | boolean  | false         |
| grpc.middleware.rateLimit.ignore          | The paths of prefix that will be ignored by middleware               | []string | []            |
| grpc.middleware.rateLimit.algorithm       | Provide algorithm, tokenBucket and leakyBucket are available options | string   | tokenBucket   |
| grpc.middleware.rateLimit.reqPerSec       | Request per second globally                                          | int      | 0             |
| grpc.middleware.rateLimit.paths.path      | Full path                                                            | string   | ""            |
| grpc.middleware.rateLimit.paths.reqPerSec | Request per second by full path                                      | int      | 0             |

#### Timeout
| name                                    | description                                            | type     | default value |
|-----------------------------------------|--------------------------------------------------------|----------|---------------|
| grpc.middleware.timeout.enabled         | Enable timeout middleware                              | boolean  | false         |
| grpc.middleware.timeout.ignore          | The paths of prefix that will be ignored by middleware | []string | []            |
| grpc.middleware.timeout.timeoutMs       | Global timeout in milliseconds.                        | int      | 5000          |
| grpc.middleware.timeout.paths.path      | Full path                                              | string   | ""            |
| grpc.middleware.timeout.paths.timeoutMs | Timeout in milliseconds by full path                   | int      | 5000          |

#### CORS
Middleware for [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

| name                                  | description                                                            | type     | default value        |
|---------------------------------------|------------------------------------------------------------------------|----------|----------------------|
| grpc.middleware.cors.enabled          | Enable cors middleware                                                 | boolean  | false                |
| grpc.middleware.cors.ignore           | The paths of prefix that will be ignored by middleware                 | []string | []                   |
| grpc.middleware.cors.allowOrigins     | Provide allowed origins with wildcard enabled.                         | []string | *                    |
| grpc.middleware.cors.allowMethods     | Provide allowed methods returns as response header of OPTIONS request. | []string | All http methods     |
| grpc.middleware.cors.allowHeaders     | Provide allowed headers returns as response header of OPTIONS request. | []string | Headers from request |
| grpc.middleware.cors.allowCredentials | Returns as response header of OPTIONS request.                         | bool     | false                |
| grpc.middleware.cors.exposeHeaders    | Provide exposed headers returns as response header of OPTIONS request. | []string | ""                   |
| grpc.middleware.cors.maxAge           | Provide max age returns as response header of OPTIONS request.         | int      | 0                    |

#### JWT
> rk-grpc using github.com/golang-jwt/jwt/v4, please beware of version compatibility.

In order to make swagger UI and RK tv work under JWT without JWT token, we need to ignore prefixes of paths as bellow.

```yaml
jwt:
  ...
  ignore:
    - "/sw"
```

| name                                          | description                                                                      | type     | default value          |
|-----------------------------------------------|----------------------------------------------------------------------------------|----------|------------------------|
| grpc.middleware.jwt.enabled                   | Optional, Enable JWT middleware                                                  | boolean  | false                  |
| grpc.middleware.jwt.ignore                    | Optional, Provide ignoring path prefix.                                          | []string | []                     |
| grpc.middleware.jwt.signerEntry               | Optional, Provide signerEntry name.                                              | string   | ""                     |
| grpc.middleware.jwt.symmetric.algorithm       | Required if symmetric specified. One of HS256, HS384, HS512                      | string   | ""                     |
| grpc.middleware.jwt.symmetric.token           | Optional, raw token for signing and verification                                 | string   | ""                     |
| grpc.middleware.jwt.symmetric.tokenPath       | Optional, path of token file                                                     | string   | ""                     |
| grpc.middleware.jwt.asymmetric.algorithm      | Required if symmetric specified. One of RS256, RS384, RS512, ES256, ES384, ES512 | string   | ""                     |
| grpc.middleware.jwt.asymmetric.privateKey     | Optional, raw private key file for signing                                       | string   | ""                     |
| grpc.middleware.jwt.asymmetric.privateKeyPath | Optional, private key file path for signing                                      | string   | ""                     |
| grpc.middleware.jwt.asymmetric.publicKey      | Optional, raw public key file for verification                                   | string   | ""                     |
| grpc.middleware.jwt.asymmetric.publicKeyPath  | Optional, public key file path for verification                                  | string   | ""                     |
| grpc.middleware.jwt.tokenLookup               | Provide token lookup scheme, please see bellow description.                      | string   | "header:Authorization" |
| grpc.middleware.jwt.authScheme                | Provide auth scheme.                                                             | string   | Bearer                 |

The supported scheme of **tokenLookup**

```
// Optional. Default value "header:Authorization".
// Possible values:
// - "header:<name>"
// - "query:<name>"
// Multiply sources example:
// - "header: Authorization,cookie: myowncookie"
```

#### Secure
Middleware for [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

| name                                         | description                                       | type     | default value   |
|----------------------------------------------|---------------------------------------------------|----------|-----------------|
| grpc.middleware.secure.enabled               | Enable secure middleware                          | boolean  | false           |
| grpc.middleware.secure.ignore                | Ignoring path prefix.                             | []string | []              |
| grpc.middleware.secure.xssProtection         | X-XSS-Protection header value.                    | string   | "1; mode=block" |
| grpc.middleware.secure.contentTypeNosniff    | X-Content-Type-Options header value.              | string   | nosniff         |
| grpc.middleware.secure.xFrameOptions         | X-Frame-Options header value.                     | string   | SAMEORIGIN      |
| grpc.middleware.secure.hstsMaxAge            | Strict-Transport-Security header value.           | int      | 0               |
| grpc.middleware.secure.hstsExcludeSubdomains | Excluding subdomains of HSTS.                     | bool     | false           |
| grpc.middleware.secure.hstsPreloadEnabled    | Enabling HSTS preload.                            | bool     | false           |
| grpc.middleware.secure.contentSecurityPolicy | Content-Security-Policy header value.             | string   | ""              |
| grpc.middleware.secure.cspReportOnly         | Content-Security-Policy-Report-Only header value. | bool     | false           |
| grpc.middleware.secure.referrerPolicy        | Referrer-Policy header value.                     | string   | ""              |

#### CSRF
Middleware for [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

| name                                | description                                                                     | type     | default value         |
|-------------------------------------|---------------------------------------------------------------------------------|----------|-----------------------|
| grpc.middleware.csrf.enabled        | Enable csrf middleware                                                          | boolean  | false                 |
| grpc.middleware.csrf.ignore         | Ignoring path prefix.                                                           | []string | []                    |
| grpc.middleware.csrf.tokenLength    | Provide the length of the generated token.                                      | int      | 32                    |
| grpc.middleware.csrf.tokenLookup    | Provide csrf token lookup rules, please see code comments for details.          | string   | "header:X-CSRF-Token" |
| grpc.middleware.csrf.cookieName     | Provide name of the CSRF cookie. This cookie will store CSRF token.             | string   | _csrf                 |
| grpc.middleware.csrf.cookieDomain   | Domain of the CSRF cookie.                                                      | string   | ""                    |
| grpc.middleware.csrf.cookiePath     | Path of the CSRF cookie.                                                        | string   | ""                    |
| grpc.middleware.csrf.cookieMaxAge   | Provide max age (in seconds) of the CSRF cookie.                                | int      | 86400                 |
| grpc.middleware.csrf.cookieHttpOnly | Indicates if CSRF cookie is HTTP only.                                          | bool     | false                 |
| grpc.middleware.csrf.cookieSameSite | Indicates SameSite mode of the CSRF cookie. Options: lax, strict, none, default | string   | default               |

### Full YAML
```yaml
---
#app:
#  name: my-app                                            # Optional, default: "rk-app"
#  version: "v1.0.0"                                       # Optional, default: "v0.0.0"
#  description: "this is description"                      # Optional, default: ""
#  keywords: ["rk", "golang"]                              # Optional, default: []
#  homeUrl: "http://example.com"                           # Optional, default: ""
#  docsUrl: ["http://example.com"]                         # Optional, default: []
#  maintainers: ["rk-dev"]                                 # Optional, default: []
#logger:
#  - name: my-logger                                       # Required
#    description: "Description of entry"                   # Optional
#    domain: "*"                                           # Optional, default: "*"
#    zap:                                                  # Optional
#      level: info                                         # Optional, default: info
#      development: true                                   # Optional, default: true
#      disableCaller: false                                # Optional, default: false
#      disableStacktrace: true                             # Optional, default: true
#      encoding: console                                   # Optional, default: console
#      outputPaths: ["stdout"]                             # Optional, default: [stdout]
#      errorOutputPaths: ["stderr"]                        # Optional, default: [stderr]
#      encoderConfig:                                      # Optional
#        timeKey: "ts"                                     # Optional, default: ts
#        levelKey: "level"                                 # Optional, default: level
#        nameKey: "logger"                                 # Optional, default: logger
#        callerKey: "caller"                               # Optional, default: caller
#        messageKey: "msg"                                 # Optional, default: msg
#        stacktraceKey: "stacktrace"                       # Optional, default: stacktrace
#        skipLineEnding: false                             # Optional, default: false
#        lineEnding: "\n"                                  # Optional, default: \n
#        consoleSeparator: "\t"                            # Optional, default: \t
#      sampling:                                           # Optional, default: nil
#        initial: 0                                        # Optional, default: 0
#        thereafter: 0                                     # Optional, default: 0
#      initialFields:                                      # Optional, default: empty map
#        key: value
#    lumberjack:                                           # Optional, default: nil
#      filename:
#      maxsize: 1024                                       # Optional, suggested: 1024 (MB)
#      maxage: 7                                           # Optional, suggested: 7 (day)
#      maxbackups: 3                                       # Optional, suggested: 3 (day)
#      localtime: true                                     # Optional, suggested: true
#      compress: true                                      # Optional, suggested: true
#    loki:
#      enabled: true                                       # Optional, default: false
#      addr: localhost:3100                                # Optional, default: localhost:3100
#      path: /loki/api/v1/push                             # Optional, default: /loki/api/v1/push
#      username: ""                                        # Optional, default: ""
#      password: ""                                        # Optional, default: ""
#      maxBatchWaitMs: 3000                                # Optional, default: 3000
#      maxBatchSize: 1000                                  # Optional, default: 1000
#      insecureSkipVerify: false                           # Optional, default: false
#      labels:                                             # Optional, default: empty map
#        my_label_key: my_label_value
#event:
#  - name: my-event                                        # Required
#    description: "Description of entry"                   # Optional
#    domain: "*"                                           # Optional, default: "*"
#    encoding: console                                     # Optional, default: console
#    outputPaths: ["stdout"]                               # Optional, default: [stdout]
#    lumberjack:                                           # Optional, default: nil
#      filename:
#      maxsize: 1024                                       # Optional, suggested: 1024 (MB)
#      maxage: 7                                           # Optional, suggested: 7 (day)
#      maxbackups: 3                                       # Optional, suggested: 3 (day)
#      localtime: true                                     # Optional, suggested: true
#      compress: true                                      # Optional, suggested: true
#    loki:
#      enabled: true                                       # Optional, default: false
#      addr: localhost:3100                                # Optional, default: localhost:3100
#      path: /loki/api/v1/push                             # Optional, default: /loki/api/v1/push
#      username: ""                                        # Optional, default: ""
#      password: ""                                        # Optional, default: ""
#      maxBatchWaitMs: 3000                                # Optional, default: 3000
#      maxBatchSize: 1000                                  # Optional, default: 1000
#      insecureSkipVerify: false                           # Optional, default: false
#      labels:                                             # Optional, default: empty map
#        my_label_key: my_label_value
#cert:
#  - name: my-cert                                         # Required
#    description: "Description of entry"                   # Optional, default: ""
#    domain: "*"                                           # Optional, default: "*"
#    caPath: "certs/ca.pem"                                # Optional, default: ""
#    certPemPath: "certs/server-cert.pem"                  # Optional, default: ""
#    keyPemPath: "certs/server-key.pem"                    # Optional, default: ""
#config:
#  - name: my-config                                       # Required
#    description: "Description of entry"                   # Optional, default: ""
#    domain: "*"                                           # Optional, default: "*"
##    path: "config/config.yaml"                            # Optional
#    envPrefix: ""                                         # Optional, default: ""
#    content:                                              # Optional, defualt: empty map
#      key: value
grpc:
  - name: greeter                                          # Required
    enabled: true                                          # Required
    port: 8080                                             # Required
#    description: "greeter server"                         # Optional, default: ""
#    enableReflection: true                                # Optional, default: false
#    enableRkGwOption: true                                # Optional, default: false
#    gwOption:                                             # Optional, default: nil
#      marshal:                                            # Optional, default: nil
#        multiline: false                                  # Optional, default: false
#        emitUnpopulated: false                            # Optional, default: false
#        indent: ""                                        # Optional, default: false
#        allowPartial: false                               # Optional, default: false
#        useProtoNames: false                              # Optional, default: false
#        useEnumNumbers: false                             # Optional, default: false
#      unmarshal:                                          # Optional, default: nil
#        allowPartial: false                               # Optional, default: false
#        discardUnknown: false                             # Optional, default: false
#    noRecvMsgSizeLimit: true                              # Optional, default: false
#    certEntry: my-cert                                    # Optional, default: "", reference of cert entry declared above
#    loggerEntry: my-logger                                # Optional, default: "", reference of cert entry declared above, STDOUT will be used if missing
#    eventEntry: my-event                                  # Optional, default: "", reference of cert entry declared above, STDOUT will be used if missing
#    sw:
#      enabled: true                                       # Optional, default: false
#      path: "sw"                                          # Optional, default: "sw"
#      jsonPath: ""                                        # Optional
#      headers: ["sw:rk"]                                  # Optional, default: []
#    docs:
#      enabled: true                                       # Optional, default: false
#      path: "docs"                                        # Optional, default: "docs"
#      specPath: ""                                        # Optional
#      headers: ["sw:rk"]                                  # Optional, default: []
#      style:                                              # Optional
#        theme: "light"                                    # Optional, default: "light"
#      debug: false                                        # Optional, default: false
#    commonService:
#      enabled: true                                       # Optional, default: false
#    static:
#      enabled: true                                       # Optional, default: false
#      path: "/static"                                     # Optional, default: /static
#      sourceType: local                                   # Required, options: pkger, local
#      sourcePath: "."                                     # Required, full path of source directory
#    pprof:
#      enabled: true                                       # Optional, default: false
#      path: "/pprof"                                      # Optional, default: /pprof
#    prom:
#      enabled: true                                       # Optional, default: false
#      path: ""                                            # Optional, default: "metrics"
#      pusher:
#        enabled: false                                    # Optional, default: false
#        jobName: "greeter-pusher"                         # Required
#        remoteAddress: "localhost:9091"                   # Required
#        basicAuth: "user:pass"                            # Optional, default: ""
#        intervalMs: 10000                                 # Optional, default: 1000
#        certEntry: my-cert                                # Optional, default: "", reference of cert entry declared above
#    middleware:
#      ignore: [""]                                        # Optional, default: []
#      logging:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        loggerEncoding: "console"                         # Optional, default: "console"
#        loggerOutputPaths: ["logs/app.log"]               # Optional, default: ["stdout"]
#        eventEncoding: "console"                          # Optional, default: "console"
#        eventOutputPaths: ["logs/event.log"]              # Optional, default: ["stdout"]
#      prom:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#      auth:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        basic:
#          - "user:pass"                                   # Optional, default: []
#        apiKey:
#          - "keys"                                        # Optional, default: []
#      meta:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        prefix: "rk"                                      # Optional, default: "rk"
#      trace:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        exporter:                                         # Optional, default will create a stdout exporter
#          file:
#            enabled: true                                 # Optional, default: false
#            outputPath: "logs/trace.log"                  # Optional, default: stdout
#          jaeger:
#            agent:
#              enabled: false                              # Optional, default: false
#              host: ""                                    # Optional, default: localhost
#              port: 0                                     # Optional, default: 6831
#            collector:
#              enabled: true                               # Optional, default: false
#              endpoint: ""                                # Optional, default: http://localhost:14268/api/traces
#              username: ""                                # Optional, default: ""
#              password: ""                                # Optional, default: ""
#      rateLimit:
#        enabled: false                                    # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        algorithm: "leakyBucket"                          # Optional, default: "tokenBucket"
#        reqPerSec: 100                                    # Optional, default: 1000000
#        paths:
#          - path: "/rk/v1/healthy"                        # Optional, default: ""
#            reqPerSec: 0                                  # Optional, default: 1000000
#      timeout:
#        enabled: false                                    # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        timeoutMs: 5000                                   # Optional, default: 5000
#        paths:
#          - path: "/rk/v1/healthy"                        # Optional, default: ""
#            timeoutMs: 1000                               # Optional, default: 5000
#      jwt:
#        enabled: true                                     # Optional, default: false
#        ignore: [ "" ]                                    # Optional, default: []
#        signerEntry: ""                                   # Optional, default: ""
#        skipValidate: false                               # Optional, default: false
#        disabledSign: false                               # Optional
#        symmetric:                                        # Optional
#          algorithm: ""                                   # Required, default: ""
#          token: ""                                       # Optional, default: ""
#          tokenPath: ""                                   # Optional, default: ""
#        asymmetric:                                       # Optional
#          algorithm: ""                                   # Required, default: ""
#          privateKey: ""                                  # Optional, default: ""
#          privateKeyPath: ""                              # Optional, default: ""
#          publicKey: ""                                   # Optional, default: ""
#          publicKeyPath: ""                               # Optional, default: ""
#        tokenLookup: "header:<name>"                      # Optional, default: "header:Authorization"
#        authScheme: "Bearer"                              # Optional, default: "Bearer"
#      secure:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        xssProtection: ""                                 # Optional, default: "1; mode=block"
#        contentTypeNosniff: ""                            # Optional, default: nosniff
#        xFrameOptions: ""                                 # Optional, default: SAMEORIGIN
#        hstsMaxAge: 0                                     # Optional, default: 0
#        hstsExcludeSubdomains: false                      # Optional, default: false
#        hstsPreloadEnabled: false                         # Optional, default: false
#        contentSecurityPolicy: ""                         # Optional, default: ""
#        cspReportOnly: false                              # Optional, default: false
#        referrerPolicy: ""                                # Optional, default: ""
#      csrf:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        tokenLength: 32                                   # Optional, default: 32
#        tokenLookup: "header:X-CSRF-Token"                # Optional, default: "header:X-CSRF-Token"
#        cookieName: "_csrf"                               # Optional, default: _csrf
#        cookieDomain: ""                                  # Optional, default: ""
#        cookiePath: ""                                    # Optional, default: ""
#        cookieMaxAge: 86400                               # Optional, default: 86400
#        cookieHttpOnly: false                             # Optional, default: false
#        cookieSameSite: "default"                         # Optional, default: "default", options: lax, strict, none, default
#      gzip:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        level: bestSpeed                                  # Optional, options: [noCompression, bestSpeed， bestCompression, defaultCompression, huffmanOnly]
#      cors:
#        enabled: true                                     # Optional, default: false
#        ignore: [""]                                      # Optional, default: []
#        allowOrigins:                                     # Optional, default: []
#          - "http://localhost:*"                          # Optional, default: *
#        allowCredentials: false                           # Optional, default: false
#        allowHeaders: []                                  # Optional, default: []
#        allowMethods: []                                  # Optional, default: []
#        exposeHeaders: []                                 # Optional, default: []
#        maxAge: 0                                         # Optional, default: 0
```

## Notice of V2
Master branch of this package is under upgrade which will be released to v2.x.x soon.

Major changes listed bellow. This will be updated with every commit.

| Last version | New version | Changes                                                                                                            |
|--------------|-------------|--------------------------------------------------------------------------------------------------------------------|
| v1.2.22      | v2          | TV is not supported because of LICENSE issue, new TV web UI will be released soon                                  |
| v1.2.22      | v2          | Remote repositry of ConfigEntry and CertEntry removed                                                              |
| v1.2.22      | v2          | Swagger json file and boot.yaml file could be embed into embed.FS and pass to rkentry                              |
| v1.2.22      | v2          | ZapLoggerEntry -> LoggerEntry                                                                                      |
| v1.2.22      | v2          | EventLoggerEntry -> EventEntry                                                                                     |
| v1.2.22      | v2          | LoggerEntry can be used as zap.Logger since all functions are inherited                                            |
| v1.2.22      | v2          | PromEntry can be used as prometheus.Registry since all functions are inherited                                     |
| v1.2.22      | v2          | rk-common dependency was removed                                                                                   |
| v1.2.22      | v2          | Entries are organized by EntryType instead of EntryName, so user can have same entry name with different EntryType |
| v1.2.22      | v2          | grpc.interceptors -> gin.middleware in boot.yaml                                                                   |
| v1.2.22      | v2          | grpc.interceptors.loggingZap -> gin.middleware.logging in boot.yaml                                                |
| v1.2.22      | v2          | grpc.interceptors.metricsProm -> gin.middleware.prom in boot.yaml                                                  |
| v1.2.22      | v2          | grpc.interceptors.tracingTelemetry -> gin.middleware.trace in boot.yaml                                            |
| v1.2.22      | v2          | All middlewares are now support gin.middleware.xxx.ignorePrefix options in boot.yaml                               |
| v1.2.22      | v2          | Middlewares support gin.middleware.ignorePrefix in boot.yaml as global scope                                       |
| v1.2.22      | v2          | LoggerEntry, EventEntry, ConfigEntry, CertEntry now support locale to distinguish in differerent environment       |
| v1.2.22      | v2          | LoggerEntry, EventEntry, CertEntry can be referenced to gin entry in boot.yaml                                     |
| v1.2.22      | v2          | Healthy API was replaced by Ready and Alive which also provides validation func from user                          |
| v1.2.22      | v2          | DocsEntry was added into rk-entry                                                                                  |
| v1.2.22      | v2          | rk-entry support utility functions of embed.FS                                                                     |
| v1.2.22      | v2          | rk-entry bumped up to v2                                                                                           |

## Development Status: Stable

## Build instruction
Simply run make all to validate your changes. Or run codes in example/ folder.

- make all

Run unit-test, golangci-lint, doctoc and gofmt.

- make buf

## Test instruction
Run unit test with **make test** command.

github workflow will automatically run unit test and golangci-lint for testing and lint validation.

## Contributing
We encourage and support an active, healthy community of contributors;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The rk maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
lark@rkdev.info.

Released under the [Apache 2.0 License](LICENSE).