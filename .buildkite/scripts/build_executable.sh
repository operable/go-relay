#!/bin/bash

set -xeuo pipefail

image="operable/relay:builder-${BUILDKITE_BUILD_NUMBER}-${BUILDKITE_COMMIT}"

echo "--- :docker: Building ${image}"
docker build -t "$image" -f Dockerfile.builder .

executable_name="relay-${BUILDKITE_BUILD_NUMBER}-${BUILDKITE_COMMIT}"
echo "--- Extracting executable as ${executable_name}"
# Need to define an entrypoint for this to work
# TODO: rework the Dockerfile to do this
container_id=$(docker create --entrypoint sh "$image")
docker cp "${container_id}:/usr/local/bin/relay" "${executable_name}"
docker rm "${container_id}"

echo "--- :buildkite: Uploading ${executable_name}"
buildkite-agent artifact upload "${executable_name}"

buildkite-agent meta-data set "executable" "${executable_name}"
