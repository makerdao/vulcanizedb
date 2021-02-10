#! /usr/bin/env bash

set -e

function message() {
    echo
    echo -----------------------------------
    echo "$@"
    echo -----------------------------------
    echo
}

ENVIRONMENT=$1
if [ "$ENVIRONMENT" == "prod" ]; then
TAG=latest
elif [ "$ENVIRONMENT" == "staging" ]; then
TAG=staging
else
   message UNKNOWN ENVIRONMENT
fi

if [ -z "$ENVIRONMENT" ]; then
    echo 'You must specifiy an environment (bash deploy.sh <ENVIRONMENT>).'
    echo 'Allowed values are "staging" or "prod"'
    exit 1
fi

# build images
COMMIT_HASH=${TRAVIS_COMMIT::7}
IMMUTABLE_TAG=$TRAVIS_BUILD_NUMBER-$COMMIT_HASH

message BUILDING HEADER-SYNC
docker build -f dockerfiles/header_sync/Dockerfile . -t makerdao/vdb-headersync:$TAG -t makerdao/vdb-headersync:$IMMUTABLE_TAG

message BUILDING RESET-HEADER-CHECK
docker build -f dockerfiles/reset_header_check_count/Dockerfile . -t makerdao/vdb-reset-header-check:$TAG -t makerdao/vdb-reset-header-check:$IMMUTABLE_TAG

message LOGGING INTO DOCKERHUB
echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USER" --password-stdin

# publish
message PUSHING HEADER-SYNC
docker image push --all-tags makerdao/vdb-headersync

message PUSHING RESET-HEADER-CHECK
docker image push --all-tags makerdao/vdb-reset-header-check

# service deploy
if [ "$ENVIRONMENT" == "prod" ]; then
  message DEPLOYING HEADER-SYNC IN PROD
  aws ecs update-service --cluster vdb-cluster-$ENVIRONMENT --service vdb-header-sync-$ENVIRONMENT --force-new-deployment --endpoint https://ecs.$PROD_REGION.amazonaws.com --region $PROD_REGION

  message DEPLOYING HEADER-SYNC IN PRIVATE-PROD
  aws ecs update-service --cluster vdb-cluster-$ENVIRONMENT --service vdb-header-sync-$ENVIRONMENT --force-new-deployment --endpoint https://ecs.$PRIVATE_PROD_REGION.amazonaws.com --region $PRIVATE_PROD_REGION

elif [ "$ENVIRONMENT" == "staging" ]; then
  message DEPLOYING HEADER-SYNC
  aws ecs update-service --cluster vdb-cluster-$ENVIRONMENT --service vdb-header-sync-$ENVIRONMENT --force-new-deployment --endpoint https://ecs.$STAGING_REGION.amazonaws.com --region $STAGING_REGION
else
   message UNKNOWN ENVIRONMENT
fi

# announce deploy
.travis/announce.sh $ENVIRONMENT vdb-header-sync
