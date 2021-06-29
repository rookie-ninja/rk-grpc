module github.com/rookie-ninja/rk-grpc-example

go 1.15

require (
	github.com/rookie-ninja/rk-entry v0.0.0-20210629191437-0d07ce6033bc
	github.com/rookie-ninja/rk-grpc v1.1.7
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.38.0
)

replace github.com/rookie-ninja/rk-grpc => ../../../
