#!/bin/bash

# Build a statically-linked Linux executable and then ensure that it
# can run on the specified platform.
#
# (We're specifically asserting here that the Relay executable can run
# on both glibc-based (e.g. Ubuntu) and musl-based (e.g. Alpine)
# Linuxes.)
#
# Intended for running in Travis CI.

# ENVIRONMENT VARIABLES:
########################################################################
#
# * TRAVIS_BUILD_ID
# * TRAVIS_COMMIT

set -euo pipefail

########################################################################
# Build the executable inside a Docker image and then extract that
# executable.

image="operable/relay:builder-${TRAVIS_BUILD_ID}-${TRAVIS_COMMIT}"
echo "--- Building ${image}"
docker build -t "$image" -f Dockerfile.builder .

executable_name="relay-${TRAVIS_BUILD_ID}-${TRAVIS_COMMIT}"
echo "--- Extracting executable as ${executable_name}"
# Need to define an entrypoint for this to work
# TODO: rework the Dockerfile to do this
container_id=$(docker create --entrypoint sh "$image")
docker cp "${container_id}:/usr/local/bin/relay" "${executable_name}"
docker rm "${container_id}"

########################################################################
# Run a minimal smoke test on the executable: try to find out the
# version of relay. We're just verifying that the executable can even
# run on this platform.

chmod a+x "${executable_name}"

for PLATFORM in alpine:3.4 ubuntu:16.10
do
    echo "--- Running ${executable_name} on ${PLATFORM}"
    set -x
    docker run \
           -it \
           --rm \
           -v $(pwd):/testing \
           "${PLATFORM}" /testing/${executable_name} --version
    set +x
done
