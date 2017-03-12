FROM gliderlabs/alpine:latest

MAINTAINER Kevin Smith <kevin@operable.io>

RUN mkdir -p /operable/cogexec/bin

COPY _build/cogexec /operable/cogexec/bin

RUN chmod +x /operable/cogexec/bin && apk-install binutils && \
    strip -s -v /operable/cogexec/bin/cogexec && apk del binutils

VOLUME /operable/cogexec