# Relay

[![Build status](https://badge.buildkite.com/6ae0ae8184e3c68a6e407b2aaa0197b24a948a01c49eb283fd.svg?branch=master)](https://buildkite.com/operable/go-relay)

## Dependencies

* Go v1.6+
* Docker v1.10.3+

## Getting up and running

1. Clone go-relay to $GOPATH/src/github.com/operable/go-relay

   ```
   mkdir -p $GOPATH/src/github.com/operable
   git clone git@github.com:operable/go-relay.git $GOPATH/src/github.com/operable
   ```

2. Download deps and compile an executable

   ```
   make
   ```

3. Set environment variables and run `relay`.

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
