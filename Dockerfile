FROM golang:1.16.3-alpine3.12 as builder

ENV CGO_ENABLED="0" \
    GOOS="linux" \
    GOARCH="amd64"

WORKDIR /go/src/github.com/h3poteto/fluentd-sidecar-injector

RUN set -ex && \
    apk add --no-cache \
    make \
    git \
    bash

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN set -ex && \
    make build

FROM alpine:latest
COPY --from=builder /go/src/github.com/h3poteto/fluentd-sidecar-injector/fluentd-sidecar-injector /fluentd-sidecar-injector

CMD ["/fluentd-sidecar-injector"]
