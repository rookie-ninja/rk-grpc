# RK gRpc interceptors
RK style gRpc interceptors which is not bind with any other frameworks.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Interceptors](#interceptors)
  - [Logging](#logging)
  - [Metrics](#metrics)
  - [Tracing](#tracing)
  - [Panic](#panic)
  - [Meta](#meta)
  - [Auth](#auth)
  - [Development Status: Stable](#development-status-stable)
  - [Appendix](#appendix)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Interceptors
### Logging
- Exchange headers between gRpc client and server.
- Server(Unary & Stream) and client(Unary & Stream) interceptors.
- Logs every gRpc requests with RPC metadata and payloads.
- Get call-scoped [zap.Logger](https://github.com/uber-go/zap/blob/master/logger.go) instance with requestId and traceId attached.
- Get call-scoped [rkquery.Event](https://github.com/rookie-ninja/rk-query/blob/master/event_zap.go) instance with RPC metadata.

### Metrics
- Exchange headers between gRpc client and server.
- Server(Unary & Stream) and client(Unary & Stream) interceptors.
- Add RequestTime, ErrorCount, ResCodeCount metrics compatible with prometheus.

### Tracing
- Exchange headers between gRpc client and server.
- Server(Unary & Stream) and client(Unary & Stream) interceptors.
- Record trace for every gRpc requests with system and RPC metadata as attributes with [opentelemetry](https://opentelemetry.io/) 
- Option to export data to stdout, files or jaeger.
- Get call-scoped Tracer, TraceSpan, TracerProvider and TracerPropagator instance.
- Create call-scoped span with utility function.

### Panic
- Catch panic errors and log stacktrace into logs.
- Recover from panic and avoid process to be killed by panic.

### Meta
- Send X-<Prefix>-<XXX> style common headers to client automatically.

### Auth
- Basic Auth
- Bearer Auth
- API Key

### Development Status: Stable

### Appendix
Use bellow command to rebuild proto files, we are using [buf](https://docs.buf.build/generate-usage) to generate proto related files.
Configuration could be found at root path of project.

- make buf
