module github.com/rookie-ninja/rk-grpc-example

go 1.15

require (
	github.com/rookie-ninja/rk-entry v0.0.0-20210630172113-abd870673153
	github.com/rookie-ninja/rk-grpc v1.1.7
	github.com/rookie-ninja/rk-prom v1.0.9-0.20210623102541-1f31500c9f12
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.38.0
)

replace github.com/rookie-ninja/rk-grpc => ../../../
