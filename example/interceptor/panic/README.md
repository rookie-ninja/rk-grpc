# Panic interceptor
In this example, we will try to create unary grpc server and client with panic interceptor enabled.

Panic interceptor will add do the bellow actions.
- Recover from panic
- Convert non status.Status().Err() to standard grpc style of error
- Set resCode to codes.Internal
- Print stacktrace
- Set [panic:1] into event as counters
- Add error into event

**Please make sure panic interceptor to be added at last in chain of interceptors.**

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Quick start](#quick-start)
  - [Code](#code)
- [Suggested panic action](#suggested-panic-action)
- [Example](#example)
  - [Start server and client](#start-server-and-client)
  - [Output](#output)
  - [Code](#code-1)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Quick start
Get rk-grpc package from the remote repository.

```go
go get -u github.com/rookie-ninja/rk-grpc
```
### Code
```go
import     "github.com/rookie-ninja/rk-grpc/interceptor/log/zap"
```
```go
    // *************************************
    // ********** Unary Server *************
    // *************************************
    opts := []grpc.ServerOption{
        grpc.ChainUnaryInterceptor(
            // Add panic interceptor at the last.
            // Please make sure panic interceptor added in the last since panic will recover() from panic
            // and add required information into logs.
            rkgrpcpanic.UnaryServerInterceptor(),
        ),
    }

    // *************************************
    // ********** Stream Server ************
    // *************************************
    opts := []grpc.ServerOption{
        grpc.ChainStreamInterceptor(
            // Add panic interceptor at the last.
            // Please make sure panic interceptor added in the last since panic will recover() from panic
            // and add required information into logs.
            rkgrpcpanic.StreamServerInterceptor(),
        ),
    }
```

## Suggested panic action
Most of the panics occurs unexpectedly. If users hope to panic by themselves, we suggest panic as bellow in order to make sure 
grpc client will receive correct error.

```go
    // Client will receive the same error as we defined.
    panic(status.Error(codes.Internal, "Panic manually!"))
```

## Example
### Start server and client
```shell script
$ go run greeter-server.go
```
```shell script
$ go run greeter-client.go
```

### Output
- Server side log (zap & event)
```shell script
2022-01-15T21:42:00.897+0800    ERROR   panic/interceptor.go:44 panic occurs:
goroutine 11 [running]:
...
        {"error": "rpc error: code = Internal desc = Panic manually!"}
------------------------------------------------------------------------
endTime=2022-01-15T21:42:00.898434+08:00
startTime=2022-01-15T21:42:00.897632+08:00
elapsedNano=802167
timezone=CST
ids={"eventId":"5d04c004-65d4-4b13-b913-f293e8ffa84f"}
app={"appName":"rk","appVersion":"","entryName":"c7hcu93d0cvkv039gghg","entryType":""}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"apiMethod":"","apiPath":"/Greeter/SayHello","apiProtocol":"","apiQuery":"","grpcMethod":"SayHello","grpcService":"Greeter","grpcType":"UnaryServer","gwMethod":"","gwPath":"","gwScheme":"","gwUserAgent":"","userAgent":""}
error={}
counters={"panic":1}
pairs={}
timing={}
remoteAddr=127.0.0.1:63002
operation=/Greeter/SayHello
resCode=Internal
eventStatus=Ended
EOE
```
- Client side log
```shell script
2022-01-15T21:42:00.899+0800    FATAL   client/greeter-client.go:31     Failed to send request to server.       {"error": "rpc error: code = Internal desc = Panic manually!"}
main.main
        /Users/dongxuny/workspace/dongxuny/rk-grpc/example/interceptor/panic/client/greeter-client.go:31
runtime.main
        /usr/local/Cellar/go/1.16.3/libexec/src/runtime/proc.go:225
```

### Code
- [greeter-server.go](server/greeter-server.go)
- [greeter-client.go](client/greeter-client.go)