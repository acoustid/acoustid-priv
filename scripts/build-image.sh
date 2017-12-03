#!/bin/sh

set -ex

if [ -n "$CI_COMMIT_TAG" ]
then
  VERSION=$(echo "$CI_COMMIT_TAG" | sed 's/^v//')
else
  VERSION=$CI_COMMIT_REF_SLUG
fi

docker build -t $CI_REGISTRY_IMAGE:$VERSION .
docker push $CI_REGISTRY_IMAGE:$VERSION

if [ -n "$CI_COMMIT_TAG" ]
then
    $(dirname $0)/tag-latest-image.sh
fi
