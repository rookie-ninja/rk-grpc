FROM golang:latest AS builder
WORKDIR /build/rk
COPY . /build/rk

RUN go env -w CGO_ENABLED="0" && go build main.go

FROM alpine

ENV WD=/usr/src/rk
WORKDIR $WD

COPY --from=builder /build/rk/main $WD/main
COPY --from=builder /build/rk/boot.yaml $WD/boot.yaml
COPY --from=builder /build/rk/api $WD/api

CMD $WD/main