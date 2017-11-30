#!/usr/bin/env bash
set -ex

mockgen -package=mock -destination=mock/repo_mock.go github.com/acoustid/priv Catalog,Repository,Account,Service
perl -pi -e 's{github.com/acoustid/priv/vendor/}{}' mock/*.go
