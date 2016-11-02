#!/bin/sh

LOOPBACK=127.0.0.1
LOCALHOST=localhost
SERVICE_HOST=${COG_SERVICE_URL_HOST}
SERVICE_PORT=${COG_SERVICE_URL_PORT}

exit_with_err()
{
  echo "Errors occured during the health check. If using the default docker-compose file from Operable, make sure COG_HOST is set correctly." 1>&2
  echo "healthcheck.sh: $@" 1>&2
  exit 1
}

is_loopback()
{
  [ $@ == $LOOPBACK -o $@ == $LOCALHOST ]
  return $?
}

# First check to see if any endpoints are set to the loopback address or localhost
if is_loopback $SERVICE_HOST
then
  exit_with_err "Can't use $LOOPBACK or $LOCALHOST for COG_SERVICE_URL_HOST."
fi

# Then check to make sure we can hit all endpoints
if ! nc -z $SERVICE_HOST $SERVICE_PORT
then
  exit_with_err "Cog Service API unavailable on $SERVICE_HOST:$SERVICE_PORT."
fi

exit 0
