#!/bin/sh

set -ex

if [ -z "$VERSION" ]
then
  if [ -n "$CI_COMMIT_TAG" ]
  then
    VERSION=$(echo "$CI_COMMIT_TAG" | sed 's/^v/')
  else
    VERSION=$CI_COMMIT_REF_SLUG
  fi
fi

docker tag $CI_REGISTRY_IMAGE:$VERSION $CI_REGISTRY_IMAGE:latest
docker push $CI_REGISTRY_IMAGE:latest
