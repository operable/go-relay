#!/bin/bash

# Download a relay executable (the name of which is stored in metadata
# under the name "executable") and see if it can actually run
# in a Docker image, specified by $IMAGE

set -euo pipefail

executable_name=$(buildkite-agent meta-data get executable)
buildkite-agent artifact download "${executable_name}" .
chmod a+x "${executable_name}"

# Just try to find out the version of relay. We're just trying to see
# that the executable can even run on this platform.
echo "--- Running ${executable_name} on ${IMAGE}"
docker run \
       -it \
       --rm \
       -v $(pwd):/testing \
       "${IMAGE}" /testing/${executable_name} --version
