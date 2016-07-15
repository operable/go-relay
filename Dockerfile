FROM alpine:3.4

MAINTAINER Kevin Smith <kevin@operable.io>

RUN mkdir -p /tmp/src/github.com/operable/go-relay/relay && mkdir -p /tmp/src/github.com/operable/go-relay/vendor
COPY .git /tmp/src/github.com/operable/go-relay
COPY main.go /tmp/src/github.com/operable/go-relay
COPY relay /tmp/src/github.com/operable/go-relay/relay
COPY vendor/vendor.json /tmp/src/github.com/operable/go-relay/vendor/vendor.json
COPY Makefile /tmp/src/github.com/operable/go-relay
COPY docker/relay.conf /usr/local/etc/relay.conf

RUN apk add -U --no-cache go make ca-certificates git && cd /tmp/src/github.com/operable/go-relay && \
  GOPATH=/tmp make exe && cp _build/relay /usr/local/bin && cd / && rm -rf /tmp/src && \
  apk del go make git
