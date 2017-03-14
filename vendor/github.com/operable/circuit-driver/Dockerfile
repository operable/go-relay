FROM gliderlabs/alpine:3.4

MAINTAINER Kevin Smith <kevin@operable.io>

RUN mkdir -p /operable/circuit/bin

COPY _build/circuit-driver /operable/circuit/bin

RUN chmod +x /operable/circuit/bin && apk-install binutils && \
    strip -s -v /operable/circuit/bin/circuit-driver && apk del binutils

VOLUME /operable/circuit