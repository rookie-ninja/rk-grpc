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

    // ************************************
    // ********** Unary Client ************
    // ************************************
    opts := []grpc.DialOption{
        grpc.WithChainUnaryInterceptor(
            // Add panic interceptor at the last.
            // Please make sure panic interceptor added in the last since panic will recover() from panic
            // and add required information into logs.
            rkgrpcpanic.UnaryClientInterceptor(),
        ),
        grpc.WithInsecure(),
        grpc.WithBlock(),
    }

    // *************************************
    // ********** Stream Client ************
    // *************************************
    opts := []grpc.DialOption{
        grpc.WithChainStreamInterceptor(
            // Add panic interceptor at the last.
            // Please make sure panic interceptor added in the last since panic will recover() from panic
            // and add required information into logs.
            rkgrpcpanic.StreamClientInterceptor(),
        ),
        grpc.WithInsecure(),
        grpc.WithBlock(),
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
2021-06-23T02:03:18.170+0800    ERROR   panic/interceptor.go:79 panic occurs:
goroutine 9 [running]:
...
main.(*GreeterServer).SayHello(0x50c9aa0, 0x4b77140, 0xc0003d6270, 0xc000328000, 0x50c9aa0, 0x0, 0x4a6a2ad)
        /Users/dongxuny/workspace/rk/rk-grpc/example/interceptor/panic/greeter-server.go:51 +0x8a
...
        {"error": "rpc error: code = Internal desc = Panic manually!"}
```
```shell script
------------------------------------------------------------------------
endTime=2021-06-23T02:03:18.171482+08:00
startTime=2021-06-23T02:03:18.170654+08:00
elapsedNano=828415
timezone=CST
ids={"eventId":"7002dd70-ccd8-440d-b392-03cd47569363"}
app={"appName":"rk","appVersion":"v0.0.0","entryName":"grpc","entryType":"grpc"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"SayHello","grpcService":"Greeter","grpcType":"unaryServer","gwMethod":"","gwPath":"","gwScheme":"","gwUserAgent":""}
error={"rpc error: code = Internal desc = Panic manually!":1}
counters={"panic":1}
pairs={}
timing={}
remoteAddr=localhost:49274
operation=/Greeter/SayHello
resCode=Internal
eventStatus=Ended
EOE
```
- Client side log (zap & event)
```shell script
2021-06-23T02:03:18.172+0800    FATAL   panic/greeter-client.go:43      Failed to send request to server.       {"error": "rpc error: code = Internal desc = Panic manually!"}
```
```shell script
------------------------------------------------------------------------
endTime=2021-06-23T02:03:18.172056+08:00
startTime=2021-06-23T02:03:18.168054+08:00
elapsedNano=4001971
timezone=CST
ids={"eventId":"b0b5c858-934d-49ae-9e04-f6985912c3b1"}
app={"appName":"rk","appVersion":"v0.0.0","entryName":"grpc","entryType":"grpc"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"SayHello","grpcService":"Greeter","grpcType":"unaryClient","remoteIp":"localhost","remotePort":"8080"}
error={"rpc error: code = Internal desc = Panic manually!":1}
counters={}
pairs={}
timing={}
remoteAddr=localhost:8080
operation=/Greeter/SayHello
resCode=Internal
eventStatus=Ended
EOE
```

### Code
- [greeter-server.go](greeter-server.go)
- [greeter-client.go](greeter-client.go)