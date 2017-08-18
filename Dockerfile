FROM alpine:3.4

MAINTAINER Christopher Maier <christopher.maier@gmail.com>

ENV GO_PACKAGE_VERSION 1.6.3-r0
ENV GOPATH /gopath
ENV PATH=${GOPATH}/bin:${PATH}

WORKDIR /gopath/src/github.com/operable/go-relay
COPY . /gopath/src/github.com/operable/go-relay

RUN apk -U add --virtual .build_deps \
    go=$GO_PACKAGE_VERSION \
    go-tools=$GO_PACKAGE_VERSION \
    git make && \

    go get -u github.com/kardianos/govendor && \
    go get -u github.com/spf13/pflag && \
    make exe && \

    mv _build/relay /usr/local/bin && \
    mkdir -p /usr/local/etc && \
    cp docker/relay.conf /usr/local/etc/relay.conf && \

    # Provide a place to dump log files, etc.
    mkdir -p /var/operable/relay && \

    apk del .build_deps && \
    rm -Rf /var/cache/apk/* && \
    rm -Rf $GOPATH
