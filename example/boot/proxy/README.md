<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [gRPC Proxy](#grpc-proxy)
  - [Example](#example)
    - [Proxy server at 8080](#proxy-server-at-8080)
    - [Test server at 8081](#test-server-at-8081)
    - [gRPC client call port of 8080](#grpc-client-call-port-of-8080)
    - [Run](#run)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# gRPC Proxy
This is under experimental.

rk-boot will proxy request if not implemented with configuration in boot.yaml.

## Example

### Proxy server at 8080
There is no gRPC API defined. Proxy request to localhost:8081 if metadata has K/V as "domain:test".

```yaml
---
grpc:
  - name: greeter                     # Required
    port: 8080                        # Required
    enabled: true                     # Required
    proxy:
      enabled: true
      rules:
        - type: headerBased
          headerPairs: ["domain:test"]
          dest: ["localhost:8081"]
#        - type: pathBased
#          paths: [""]
#          dest: [""]
#        - type: IpBased
#          Ips: [""]
#          dest: [""]
```

- [main.go](proxy/main.go)

### Test server at 8081
Enable common service in order to receive proxied request from 8080.

```yaml
---
grpc:
  - name: greeter                     # Required
    port: 8081                        # Required
    enabled: true                     # Required
    commonService:
      enabled: true                   # Optional, default: false
```

- [main.go](test/main.go)

### gRPC client call port of 8080
Currently, proxy is only supported with gRPC client with codes. 

grpc-gateway or grpcurl is not supported for proxying.

- [main.go](client/main.go)

### Run
- Run proxy server
```shell script
go run proxy/main.go
```

- Run test server
```shell script
go run test/main.go
```

- Run client
```shell script
go run client/main.go

2021-11-12T22:39:45.516+0800    INFO    client/greeter-client.go:45     [Message]: fields:{key:"healthy" value:{bool_value:true}}
```