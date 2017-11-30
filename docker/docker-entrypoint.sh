#!/usr/bin/env bash

: ${ACOUSTID_PRIV_BIND:=0.0.0.0:5000}
: ${ACOUSTID_PRIV_DB_URL:=postgresql://localhost/acoustid_priv}

args=( -bind=$ACOUSTID_PRIV_BIND -db=$ACOUSTID_PRIV_DB_URL )

exec "$@" "${args[@]}"