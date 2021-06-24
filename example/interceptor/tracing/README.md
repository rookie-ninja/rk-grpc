# Trace interceptor
In this example, we will try to create unary grpc server and client with trace interceptor enabled.

Trace interceptor has bellow options currently while exporting tracing information.

| Exporter | Description |
| ---- | ---- |
| Stdout | Export as JSON style. |
| Local file | Export as JSON style. |
| Jaeger |  In beta stage, export to jaeger collector only. |

**Please make sure panic interceptor to be added at last in chain of interceptors.**

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Quick start](#quick-start)
- [Options](#options)
  - [Exporter](#exporter)
    - [Stdout exporter](#stdout-exporter)
    - [File exporter](#file-exporter)
    - [Jaeger exporter](#jaeger-exporter)
- [Example](#example)
  - [Start server and client](#start-server-and-client)
  - [Output](#output)
    - [Stdout exporter](#stdout-exporter-1)
    - [Jaeger exporter](#jaeger-exporter-1)
  - [Code](#code)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Quick start
Get rk-grpc package from the remote repository.

```go
go get -u github.com/rookie-ninja/rk-grpc
```
```go
    // *************************************
    // ********** Unary Server *************
    // *************************************
    opts := []grpc.ServerOption{
        grpc.ChainUnaryInterceptor(
            // Add trace interceptor
            rkgrpctrace.UnaryServerInterceptor(
                // Entry name and entry type will be used for distinguishing interceptors. Recommended.
                // rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
                //
                // Provide an exporter.
                // rkgrpctrace.WithExporter(exporter),
                //
                // Provide propagation.TextMapPropagator
                // rkgrpctrace.WithPropagator(<propagator>),
                // 
                // Provide SpanProcessor
                // rkgrpctrace.WithSpanProcessor(<span processor>),
                // 
                // Provide TracerProvider
                // rkgrpctrace.WithTracerProvider(<trace provider>),
            ),
        ),
    }

    // *************************************
    // ********** Stream Server ************
    // *************************************
    opts := []grpc.ServerOption{
        grpc.ChainStreamInterceptor(
            // Add trace interceptor
            rkgrpctrace.StreamServerInterceptor(
                // Entry name and entry type will be used for distinguishing interceptors. Recommended.
                // rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
                //
                // Provide an exporter.
                // rkgrpctrace.WithExporter(exporter),
                //
                // Provide propagation.TextMapPropagator
                // rkgrpctrace.WithPropagator(<propagator>),
                // 
                // Provide SpanProcessor
                // rkgrpctrace.WithSpanProcessor(<span processor>),
                // 
                // Provide TracerProvider
                // rkgrpctrace.WithTracerProvider(<trace provider>),
            ),
        ),
    }

    // ************************************
    // ********** Unary Client ************
    // ************************************
    opts := []grpc.DialOption{
        grpc.WithChainUnaryInterceptor(
            // Add trace interceptor
            rkgrpctrace.UnaryClientInterceptor(
                // Entry name and entry type will be used for distinguishing interceptors. Recommended.
                // rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
                //
                // Provide an exporter.
                // rkgrpctrace.WithExporter(exporter),
                //
                // Provide propagation.TextMapPropagator
                // rkgrpctrace.WithPropagator(<propagator>),
                // 
                // Provide SpanProcessor
                // rkgrpctrace.WithSpanProcessor(<span processor>),
                // 
                // Provide TracerProvider
                // rkgrpctrace.WithTracerProvider(<trace provider>),
            ),
        ),
        grpc.WithInsecure(),
        grpc.WithBlock(),
    }

    // *************************************
    // ********** Stream Client ************
    // *************************************
    opts := []grpc.DialOption{
        grpc.WithChainStreamInterceptor(
            // Add trace interceptor
            rkgrpctrace.StreamClientInterceptor(
                // Entry name and entry type will be used for distinguishing interceptors. Recommended.
                // rkgrpclog.WithEntryNameAndType("greeter", "grpc"),
                //
                // Provide an exporter.
                // rkgrpctrace.WithExporter(exporter),
                //
                // Provide propagation.TextMapPropagator
                // rkgrpctrace.WithPropagator(<propagator>),
                // 
                // Provide SpanProcessor
                // rkgrpctrace.WithSpanProcessor(<span processor>),
                // 
                // Provide TracerProvider
                // rkgrpctrace.WithTracerProvider(<trace provider>),
            ),
        ),
        grpc.WithInsecure(),
        grpc.WithBlock(),
    }
```

## Options
If client didn't enable trace interceptor, then server will create a new trace span by itself. If client sends a tracemeta to server, 
then server will use the same traceId.

| Name | Description | Default |
| ---- | ---- | ---- |
| WithEntryNameAndType(entryName, entryType string) | Provide entryName and entryType, recommended. | entryName=grpc, entryType=grpc |
| WithExporter(exporter sdktrace.SpanExporter) | User defined exporter. | [Stdout exporter](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/stdout) with pretty print and disabled metrics |
| WithSpanProcessor(processor sdktrace.SpanProcessor) | User defined span processor. | [NewBatchSpanProcessor](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace#NewBatchSpanProcessor) |
| WithPropagator(propagator propagation.TextMapPropagator) | User defined propagator. | [NewCompositeTextMapPropagator](https://pkg.go.dev/go.opentelemetry.io/otel/propagation#TextMapPropagator) |

![server-arch](img/server-arch.png)
![client-arch](img/client-arch.png)

### Exporter
#### Stdout exporter
```go
    // ****************************************
    // ********** Create Exporter *************
    // ****************************************

    // Export trace to stdout with utility function
    //
    // Bellow function would be while creation
    // set.Exporter, _ = stdout.NewExporter(
    //     stdout.WithPrettyPrint(),
    //     stdout.WithoutMetricExport())
    exporter := rkgrpctrace.CreateFileExporter("stdout")

    // Users can define own stdout exporter by themselves.
    exporter, _ := stdout.NewExporter(stdout.WithPrettyPrint(), stdout.WithoutMetricExport())
```

#### File exporter
```go
    // ****************************************
    // ********** Create Exporter *************
    // ****************************************

    // Export trace to local file system
    exporter := rkgrpctrace.CreateFileExporter("logs/trace.log")
```

#### Jaeger exporter
```go
    // ****************************************
    // ********** Create Exporter *************
    // ****************************************

    // Export trace to jaeger collector
    exporter := rkgrpctrace.CreateJaegerExporter("localhost:14368", "", "")
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
#### Stdout exporter
If logger interceptor enabled, then traceId would be attached to event and zap logger.

- Server side trace log
```shell script
[
        {
                "SpanContext": {
                        "TraceID": "898ad20ad69998dc0bef2707ce5332d5",
                        "SpanID": "1a163e6bb2f96dcd",
                        "TraceFlags": "01",
                        "TraceState": null,
                        "Remote": false
                },
                ...
```

- Server side log (zap & event)
```shell script
2021-06-23T16:53:21.669+0800    INFO    tracing/greeter-server.go:59    Received client request!        {"traceId": "898ad20ad69998dc0bef2707ce5332d5"}
```
```shell script
------------------------------------------------------------------------
endTime=2021-06-23T16:53:21.669385+08:00
startTime=2021-06-23T16:53:21.669175+08:00
elapsedNano=210152
timezone=CST
ids={"eventId":"5425933f-3a41-4e2a-80df-6261c4d7aaaf","traceId":"898ad20ad69998dc0bef2707ce5332d5"}
app={"appName":"rk","appVersion":"v0.0.0","entryName":"grpc","entryType":"grpc"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"SayHello","grpcService":"Greeter","grpcType":"unaryServer","gwMethod":"","gwPath":"","gwScheme":"","gwUserAgent":""}
error={}
counters={}
pairs={}
timing={}
remoteAddr=localhost:51192
operation=/Greeter/SayHello
resCode=OK
eventStatus=Ended
EOE
```

- Client side trace log
```shell script
[
        {
                "SpanContext": {
                        "TraceID": "898ad20ad69998dc0bef2707ce5332d5",
                        "SpanID": "c040f9e3771ab5fc",
                        "TraceFlags": "01",
                        "TraceState": null,
                        "Remote": false
                },
                ...
```

- Client side log (zap & event)
```shell script
2021-06-23T16:53:21.670+0800    INFO    tracing/greeter-client.go:58    [Message]: Hello rk-dev!        {"traceId": "898ad20ad69998dc0bef2707ce5332d5"}
```
```shell script
------------------------------------------------------------------------
endTime=2021-06-23T16:53:21.670053+08:00
startTime=2021-06-23T16:53:21.667635+08:00
elapsedNano=2418516
timezone=CST
ids={"eventId":"351b9e71-7cc4-4631-8050-a876a8dd0809","traceId":"898ad20ad69998dc0bef2707ce5332d5"}
app={"appName":"rk","appVersion":"v0.0.0","entryName":"grpc","entryType":"grpc"}
env={"arch":"amd64","az":"*","domain":"*","hostname":"lark.local","localIP":"10.8.0.2","os":"darwin","realm":"*","region":"*"}
payloads={"grpcMethod":"SayHello","grpcService":"Greeter","grpcType":"unaryClient","remoteIp":"localhost","remotePort":"8080"}
error={}
counters={}
pairs={}
timing={}
remoteAddr=localhost:8080
operation=/Greeter/SayHello
resCode=OK
eventStatus=Ended
EOE
```

#### Jaeger exporter
![Jaeger](img/jaeger.png)

### Code
- [greeter-server.go](greeter-server.go)
- [greeter-client.go](greeter-client.go)