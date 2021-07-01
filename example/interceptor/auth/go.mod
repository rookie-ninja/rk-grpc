module github.com/rookie-ninja/rk-grpc-example

go 1.15

require (
	github.com/rookie-ninja/rk-entry v0.0.0-20210630180059-341e0517e5a2
	github.com/rookie-ninja/rk-grpc v1.1.7
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.38.0
)

replace github.com/rookie-ninja/rk-grpc => ../../../
