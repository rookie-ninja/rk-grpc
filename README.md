<h2 align="center">
  rk-grpc
</h2>
<p align="center">
  Inject middlewares & server configuration of <a href="https://grpc.io/docs/languages/go/">gRPC</a> and <a href="https://github.com/grpc-ecosystem/grpc-gateway">grpc-gateway</a> from YAML file.
</p>
<p align="center">
  This belongs to <a href="https://github.com/rookie-ninja/rk-boot">rk-boot</a> family. We suggest use this lib with <a href="https://github.com/rookie-ninja/rk-boot">rk-boot</a>.
</p>

<p align="center">
 <a href="https://github.com/rookie-ninja/rk-grpc/actions/workflows/ci.yml"><img src="https://github.com/rookie-ninja/rk-grpc/actions/workflows/ci.yml/badge.svg"></a>
 <a href="https://codecov.io/gh/rookie-ninja/rk-grpc"><img src="https://codecov.io/gh/rookie-ninja/rk-grpc/branch/master/graph/badge.svg?token=08TCFIIVS0"></a>
 <a href="https://goreportcard.com/badge/github.com/rookie-ninja/rk-grpc"><img src="https://goreportcard.com/badge/github.com/rookie-ninja/rk-grpc"></a>
 <a href="https://sourcegraph.com/github.com/rookie-ninja/rk-grpc?badge"><img src="https://sourcegraph.com/github.com/rookie-ninja/rk-grpc/-/badge.svg"></a>
 <a href="https://godoc.org/github.com/rookie-ninja/rk-grpc"><img src="https://godoc.org/github.com/rookie-ninja/rk-grpc?status.svg"></a>
 <a href="https://github.com/rookie-ninja/rk-grpc/releases"><img src="https://img.shields.io/github/release/rookie-ninja/rk-grpc.svg?style=flat-square"></a>
 <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg"></a>
<p>

<div id="badges" align="center">
  <a href="https://medium.com/@pointgoal/list/grpc-101-790c4c160a05">
    <img src="https://img.shields.io/badge/Medium-12100E?style=for-the-badge&logo=medium&logoColor=white" alt="Medium Badge"/>
  </a>
  <a href="https://rkdev.info">
    <img src="https://img.shields.io/badge/Official Site-blue?logo=mdbook&logoColor=white&style=for-the-badge" alt="Docs Badge"/>
  </a>
  <a href="https://rk-syz1767.slack.com/rk-boot">
    <img src="https://img.shields.io/badge/Slack-4A154B?style=for-the-badge&logo=slack&logoColor=white" alt="Docs Badge"/>
  </a>
</div>


## Architecture
![image](docs/img/grpc-arch.png)

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

### Installation

```shell
go get github.com/rookie-ninja/rk-grpc/v2
```

### 1.Prepare .proto files
<details>
<summary>show</summary>

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
</details>

### 2.Generate .pb.go files with [buf](https://docs.buf.build/introduction)
<details>
<summary>show</summary>

```
$ buf generate --path api/v1
```

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
</details>

### 3.Create boot.yaml
<details>
<summary>show</summary>

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
</details>

### 4.Create main.go
<details>
<summary>show</summary>

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

</details>

### 5.Start server
```
$ go run main.go
```

### 6.Validation
<details>
<summary>show</summary>

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

#### 6.3 Docs UI
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

</details>

## Supported features
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

## YAML options
User can start multiple [gRPC](https://grpc.io/docs/languages/go/) and [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) instances at the same time. Please make sure use different port and name.

<details>
<summary>show</summary>

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
#    default: false                                        # Optional, default: false, use as default logger entry
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
#    default: false                                        # Optional, default: false, use as default event entry
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
#    grpcWeb:
#      enabled: true
#      cors:
#        allowOrigins: []                                  # Optional, default: [*]
#      websocket:
#        enabled: true                                     # Optional, default: disable websocket
#        pingIntervalMs: 10                                # Optional, default: disable ping
#        messageReadLimitBytes: 32769                      # Optional, default: 32769
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
#      errorModel: google                                  # Optional, default: google, [amazon, google] are supported options
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
#        skipVerify: false                                 # Optional, default: false
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

</details>

## Development Status: Stable

## Build instruction
Simply run make all to validate your changes. Or run codes in example/ folder.

- make all

Run unit-test, golangci-lint, doctoc and gofmt.

- make buf

## Test instruction
Run unit test with **make test** command.

Github workflow will automatically run unit test and golangci-lint for testing and lint validation.

## Contributing
We encourage and support an active, healthy community of contributors;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The rk maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
lark@rkdev.info.

Released under the [Apache 2.0 License](LICENSE).