#!/usr/bin/env bash

: ${ACOUSTID_PRIV_BIND:=0.0.0.0:5000}
: ${ACOUSTID_PRIV_DB_URL:=postgresql://localhost/acoustid_priv}

if [ "${1:0:1}" = '-' ]
then
	set -- acoustid-priv-api "$@"
fi

args=()

if [ "$1" = 'acoustid-priv-api' ]
then
    args=( -bind=$ACOUSTID_PRIV_BIND -db=$ACOUSTID_PRIV_DB_URL )
fi

exec "$@" "${args[@]}"