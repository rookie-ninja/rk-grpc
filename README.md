# rk-interceptor
gRPC interceptor

- [zap](https://github.com/uber-go/zap)
- [lumberjack](https://github.com/natefinch/lumberjack)
- [rk-query](https://github.com/rookie-ninja/rk-logger)

## Installation
`go get -u rookie-ninja/rk-interceptor`

## Quick Start
An event needs to be pass into intercetpr in order to write logs

Please refer https://github.com/rookie-ninja/rk-query for easy initialization of Event

### Server side interceptor

Example:
```go
var (
	bytes = []byte(`{
     "level": "info",
     "encoding": "console",
     "outputPaths": ["stdout"],
     "errorOutputPaths": ["stderr"],
     "initialFields": {},
     "encoderConfig": {
       "messageKey": "msg",
       "levelKey": "",
       "nameKey": "",
       "timeKey": "",
       "callerKey": "",
       "stacktraceKey": "",
       "callstackKey": "",
       "errorKey": "",
       "timeEncoder": "iso8601",
       "fileKey": "",
       "levelEncoder": "capital",
       "durationEncoder": "second",
       "callerEncoder": "full",
       "nameEncoder": "full"
     },
    "maxsize": 1,
    "maxage": 7,
    "maxbackups": 3,
    "localtime": true,
    "compress": true
   }`)

	logger, _, _ = rk_logger.NewZapLoggerWithBytes(bytes, rk_logger.JSON)
)

func main() {
	// create listener
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create event factory
	factory := rk_query.NewEventFactory(
		rk_query.WithAppName("my-app"),
		rk_query.WithLogger(logger),
		rk_query.WithFormat(rk_query.RK))

	// create server interceptor
	opt := []grpc.ServerOption{
		grpc.UnaryInterceptor(rk_logging_zap.UnaryServerInterceptor(factory)),
	}

	// create server
	s := grpc.NewServer(opt...)
	proto.RegisterGreeterServer(s, &GreeterServer{})

	// serving
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type GreeterServer struct{}

func (server *GreeterServer) SayHello(ctx context.Context, request *proto.HelloRequest) (*proto.HelloResponse, error) {
	event := rk_context.GetEvent(ctx)
	// add fields
	event.AddFields(zap.String("key", "value"))
	// add error
	event.AddErr(errors.New(""))
	// add pair
	event.AddPair("key", "value")
	// set counter
	event.SetCounter("ctr", 1)
	// timer
	event.StartTimer("sleep")
	time.Sleep(1 * time.Second)
	event.EndTimer("sleep")
	// add to metadata
	rk_context.AddToOutgoingMD(ctx, "key", "1", "2")
	// add request id
	rk_context.AddRequestIdToOutgoingMD(ctx)

	// print incoming metadata
	bytes, _ := json.Marshal(rk_context.GetIncomingMD(ctx))
	println(string(bytes))

	return &proto.HelloResponse{
		Message: "hello",
	}, nil
}
```
Output
```
------------------------------------------------------------------------
end_time=2020-07-31T04:01:13.477136+08:00
start_time=2020-07-31T04:01:12.475701+08:00
time=1001
hostname=MYLOCAL
event_id=["afd6bb44-5296-42d8-8850-b526733d9f67","66d7801e-b06c-4cea-8397-391c8ffc586b"]
timing={"sleep.count":1,"sleep.elapsed_ms":1001}
counter={"ctr":1}
pair={"key":"value"}
error={"std-err":1}
field={"api.role":"unary_server","api.service":"Greeter","api.verb":"SayHello","app_version":"latest","az":"unknown","deadline":"2020-07-31T04:01:17+08:00","domain":"unknown","elapsed_ms":1001,"end_time":"2020-07-31T04:01:13.477136+08:00","incoming_request_id":["afd6bb44-5296-42d8-8850-b526733d9f67"],"key":"value","local.IP":"10.8.0.6","outgoing_request_id":["66d7801e-b06c-4cea-8397-391c8ffc586b"],"realm":"unknown","region":"unknown","remote.IP":"localhost","remote.net_type":"tcp","remote.port":"62541","res_code":"OK","start_time":"2020-07-31T04:01:12.475701+08:00"}
remote_addr=localhost
app_name=my-app
operation=SayHello
event_status=Ended
history=s-sleep:1596139272475,e-sleep:1001,end:1
EOE

```

### Client side interceptor

Example:
```go
var (
	bytes = []byte(`{
     "level": "info",
     "encoding": "console",
     "outputPaths": ["stdout"],
     "errorOutputPaths": ["stderr"],
     "initialFields": {},
     "encoderConfig": {
       "messageKey": "msg",
       "levelKey": "",
       "nameKey": "",
       "timeKey": "",
       "callerKey": "",
       "stacktraceKey": "",
       "callstackKey": "",
       "errorKey": "",
       "timeEncoder": "iso8601",
       "fileKey": "",
       "levelEncoder": "capital",
       "durationEncoder": "second",
       "callerEncoder": "full",
       "nameEncoder": "full"
     },
    "maxsize": 1,
    "maxage": 7,
    "maxbackups": 3,
    "localtime": true,
    "compress": true
   }`)

	logger, _, _ = rk_logger.NewZapLoggerWithBytes(bytes, rk_logger.JSON)
)

func main() {
	// create event factory
	factory := rk_query.NewEventFactory(
		rk_query.WithAppName("app"),
		rk_query.WithLogger(logger),
		rk_query.WithFormat(rk_query.RK))

	// create client interceptor
	opt := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			rk_logging_zap.UnaryClientInterceptor(factory)),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}

	// Set up a connection to the server.
	conn, err := grpc.DialContext(context.Background(), "localhost:8080", opt...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// create grpc client
	c := proto.NewGreeterClient(conn)
	// create with rk context
	ctx, cancel := context.WithTimeout(rk_context.NewContext(), 5*time.Second)
	defer cancel()

	// add metadata
	rk_context.AddToOutgoingMD(ctx, "key", "1", "2")
	// add request id
	rk_context.AddRequestIdToOutgoingMD(ctx)

	// call server
	r, err := c.SayHello(ctx, &proto.HelloRequest{Name: "name"})

	// print incoming metadata
	bytes, _ := json.Marshal(rk_context.GetIncomingMD(ctx))
	println(string(bytes))

	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
```
Output 
```
------------------------------------------------------------------------
end_time=2020-07-31T04:01:13.478851+08:00
start_time=2020-07-31T04:01:12.474995+08:00
time=1003
hostname=MYLOCAL
event_id=["66d7801e-b06c-4cea-8397-391c8ffc586b","afd6bb44-5296-42d8-8850-b526733d9f67"]
timing={}
counter={}
pair={}
error={}
field={"api.role":"unary_client","api.service":"Greeter","api.verb":"SayHello","app_version":"latest","az":"unknown","deadline":"2020-07-31T04:01:17+08:00","domain":"unknown","elapsed_ms":1003,"end_time":"2020-07-31T04:01:13.47886+08:00","incoming_request_id":["66d7801e-b06c-4cea-8397-391c8ffc586b"],"local.IP":"10.8.0.6","outgoing_request_id":["afd6bb44-5296-42d8-8850-b526733d9f67"],"realm":"unknown","region":"unknown","remote.IP":"localhost","remote.port":"8080","res_code":"OK","start_time":"2020-07-31T04:01:12.474995+08:00"}
remote_addr=localhost
app_name=app
operation=SayHello
event_status=Ended
EOE

```


### Development Status: Stable

### Contributing
We encourage and support an active, healthy community of contributors &mdash;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The rk maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
dongxuny@gmail.com. That email list is a private, safe space; even the zap
maintainers don't have access, so don't hesitate to hold us to a high
standard.

<hr>

Released under the [MIT License](LICENSE).

