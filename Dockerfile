FROM gliderlabs/alpine:latest

MAINTAINER Kevin Smith <kevin@operable.io>

COPY _build/relay /usr/local/bin/relay
COPY docker/relay.conf /usr/local/etc/relay.conf

# RUN chmod +x /usr/local/bin/relay

# Ensure latest CA certs are available
RUN apk-install ca-certificates && chmod +x /usr/local/bin/relay
