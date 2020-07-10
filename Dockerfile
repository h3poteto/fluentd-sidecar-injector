FROM golang:1.13.7-alpine3.11 as builder

ENV CGO_ENABLED="0" \
    GOOS="linux" \
    GOARCH="amd64"

WORKDIR /go/src/github.com/h3poteto/fluentd-sidecar-injector

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN set -ex && \
    go build -o fluentd-sidecar-injector

FROM alpine:latest
COPY --from=builder /go/src/github.com/h3poteto/fluentd-sidecar-injector /fluentd-sidecar-injector

CMD ["/fluentd-sidecar-injector"]
