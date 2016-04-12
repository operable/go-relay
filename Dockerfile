FROM gliderlabs/alpine:latest

MAINTAINER Kevin Smith <kevin@operable.io>

# Ensure latest CA certs are available
RUN apk-install ca-certificates

COPY _build/relay /usr/local/bin/relay
COPY docker/relay.conf /usr/local/etc/relay.conf

RUN chmod +x /usr/local/bin/cog-relay
