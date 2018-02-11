# Relay

[![Build Status](https://travis-ci.org/operable/go-relay.svg?branch=master)](https://travis-ci.org/operable/go-relay)
[![Coverage Status](https://coveralls.io/repos/github/operable/go-relay/badge.svg?branch=master)](https://coveralls.io/github/operable/go-relay?branch=master)
[![Ebert](https://ebertapp.io/github/operable/go-relay.svg)](https://ebertapp.io/github/operable/go-relay)
[![Docker Build Statu](https://img.shields.io/docker/build/operable/relay-preview.svg)]()

## Dependencies

* Go v1.9+
* Docker v1.10.3+

## Getting up and running

1. Clone go-relay to $GOPATH/src/github.com/operable/go-relay

   ```
   mkdir -p $GOPATH/src/github.com/operable
   git clone git@github.com:operable/go-relay.git $GOPATH/src/github.com/operable
   ```

2. Install preqrequisites

```sh
go get -u github.com/kardianos/govendor
```

3. Download deps and compile an executable

   ```
   make
   ```

4. Set environment variables and run `relay`.

   You'll need to have a docker machine running and have environment variables
   set for the docker client to connect to it. If you haven't already, run the
   following. (See more details at https://docs.docker.com/machine/get-started)

   ```
   docker-machine create --driver virtualbox default
   docker-machine start default
   eval $(docker-machine env default)`
   ```

   Then start relay:

   ```
   RELAY_DOCKER_USE_ENV=true _build/relay -file example_cog_relay.conf
   ```

## Docker Images

Release images are available from the
[operable/relay](https://hub.docker.com/r/operable/relay-preview/)
repository on Docker Hub.

The latest code from the `master` branch is always available in the
[operable/relay-preview](https://hub.docker.com/r/operable/relay-preview/)
repository. The only tag in this repository is `latest`, and it
"floats".

Building your own image can be done with `make docker`.
