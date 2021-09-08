module github.com/rookie-ninja/rk-grpc-example

go 1.15

require (
	github.com/rookie-ninja/rk-entry v1.0.3
	github.com/rookie-ninja/rk-grpc v1.1.7
	github.com/rookie-ninja/rk-prom v1.1.3
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.38.0
)

replace github.com/rookie-ninja/rk-grpc => ../../../
