FROM alpine:3.4

MAINTAINER Kevin Smith <kevin@operable.io>

# Bake in a directory that we can use for logging, config, etc.
RUN mkdir -p /var/operable/relay

# Relies on the binary having already been built
COPY _build/relay /usr/local/bin
COPY docker/relay.conf /usr/local/etc/relay.conf
