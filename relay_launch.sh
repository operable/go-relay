#!/bin/sh

########################################################
# Configure and run relay to use docker-machine        #
########################################################
usage() {
  echo "launch_relay.sh <docker-machine env name> ..."
  echo ""
  echo "NOTE: Arguments after env name are passed thru to relay untouched."
}

script_dir=`cd $(dirname ${0}) && pwd`

# First check to see if relay is in $PATH
relay_bin=`which relay`

# If not in $PATH check current dir
if [ "${relay_bin}" == "" ]; then
  if [ -f ${script_dir}/relay ]; then
    relay_bin=${script_dir}/relay
  else
    # If not in current dir see if we can
    # find a relay binary in _build
    if [ -f ${script_dir}/_build/relay ]; then
      relay_bin=${script_dir}/_build/relay
    else
      printf "'relay' binary not found.\n" 1>&2
      exit 1
    fi
  fi
fi

if [ $# > 1 ]; then
  docker_env=${1}
  shift
fi

if [ "${docker_env}" == "" ]; then
  printf "Which docker-machine environment should I use?\n" 1>&2
  usage
  exit 1
fi

eval $(docker-machine env ${docker_env})

${relay_bin} $@
