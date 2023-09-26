FROM 035494162654.dkr.ecr.cn-north-1.amazonaws.com.cn/mof-builder:latest AS builder
WORKDIR /build/mof
COPY . /build/mof

RUN go env -w CGO_ENABLED="0" && go build main.go

FROM alpine

ENV WD=/usr/src/mof
WORKDIR $WD

COPY --from=builder /build/mof/main $WD/main
COPY --from=builder /build/mof/boot.yaml $WD/boot.yaml
COPY --from=builder /build/mof/api $WD/api

CMD $WD/main