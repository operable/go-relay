#!/bin/sh

# git describe --tags appends SHA info to
# prior tag. This subcommand detects when that
# happens.
tag=`git describe --tags`
dash_count=`git describe --tags | grep -o - | wc -l | sed "s/ //g"`

if [ "$dash_count" != "0" ]; then
  tag="dev"
fi

echo $tag
