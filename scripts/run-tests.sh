#!/usr/bin/env bash

set -ex

wait-for-it.sh $ACOUSTID_PRIV_TEST_DB_HOST:5432

exec go test -v github.com/acoustid/priv/...
