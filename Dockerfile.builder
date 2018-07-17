FROM ubuntu:17.10

# This is actually Go 1.8.3, despite appearances (as determined by
# running `go version`)
ENV GO_PACKAGE_VERSION 2:1.8~1ubuntu1
ENV GOPATH /gopath
ENV PATH=${GOPATH}/bin:${PATH}

RUN apt-get update && \
    apt-get install -y \
            git \
            golang-go=$GO_PACKAGE_VERSION

RUN go get -u github.com/golang/lint/golint
RUN go get -u github.com/kardianos/govendor
RUN go get -u github.com/spf13/pflag

WORKDIR /gopath/src/github.com/operable/go-relay
COPY . /gopath/src/github.com/operable/go-relay
RUN make exe

# NOTE: This is not intended to be an Ubuntu Relay image for
# deployment, so it doesn't have a config file in place. It's just for
# building a binary.
RUN mv _build/relay /usr/local/bin
