#!/bin/bash

# Download a relay executable (the name of which is stored in metadata
# under the name "executable") and inject it into a new Docker image.

set -euo pipefail

executable_name=$(buildkite-agent meta-data get executable)
echo "--- Downloading artifact ${executable_name}"
buildkite-agent artifact download "${executable_name}" .
chmod a+x "${executable_name}"
mkdir _build
cp "${executable_name}" _build/relay

export DOCKER_IMAGE="operable/relay-testing:${BUILDKITE_BUILD_NUMBER}-${BUILDKITE_COMMIT}"
echo "--- Building image ${DOCKER_IMAGE}"
make do-docker-build

echo "--- Pushing ${DOCKER_IMAGE}"
docker push "${DOCKER_IMAGE}"
