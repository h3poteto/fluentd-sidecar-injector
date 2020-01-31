FROM golang:1.13.7-alpine3.11

WORKDIR /go/src/github.com/h3poteto/fluentd-sidecar-injector

COPY . .

RUN set -ex && \
    go build -o fluentd-sidecar-injector

CMD ["./fluentd-sidecar-injector"]
