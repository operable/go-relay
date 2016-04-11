# Relay

## Dependencies

Go v1.6+
Docker v1.10.3+

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

3. Set environment variables and run `cog-relay`.

   ```
   RELAY_DOCKER_USE_ENV=true ./cog-relay -file example_cog_relay.conf
   ```

   NOTE: You'll need to have a docker machine running and have environment
   variables set for the docker client to connect to it.
   https://docs.docker.com/machine/get-started/
