FROM gliderlabs/alpine:latest

MAINTAINER Kevin Smith <kevin@operable.io>

# Install wget
RUN apk-install curl strace ca-certificates

# Enable strace-ing
RUN echo kernel.yama.ptrace_scope=0 > /etc/sysctl.d/00-operable.conf

COPY cog-relay /usr/local/bin/cog-relay
COPY docker/cog_relay.conf /usr/local/etc/cog_relay.conf

RUN chmod +x /usr/local/bin/cog-relay

RUN apk del curl
