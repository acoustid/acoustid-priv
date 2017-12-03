#!/bin/sh

set -ex

docker login -u gitlab-ci-token -p $CI_BUILD_TOKEN $CI_REGISTRY

if [ -n "$CI_COMMIT_TAG" ]
then
  VERSION=$(echo "$CI_COMMIT_TAG" | sed 's/^v/')
else
  VERSION=$CI_COMMIT_REF_SLUG
fi

docker build -t $CI_REGISTRY_IMAGE:$VERSION .
docker push $CI_REGISTRY_IMAGE:$VERSION
