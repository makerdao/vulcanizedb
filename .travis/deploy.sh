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
#--------------------------
# INIT
#--------------------------
if [ "$ENVIRONMENT" == "prod" ]; then
    TAG=latest
    REGION=$PROD_REGION
elif [ "$ENVIRONMENT" == "private-prod" ]; then
    ENVIRONMENT="prod"
    TAG=latest
    REGION=$PRIVATE_PROD_REGION
elif [ "$ENVIRONMENT" == "staging" ]; then
    TAG=staging
    REGION=$STAGING_REGION
elif [ "$ENVIRONMENT" == "qa" ]; then
    TAG=develop
    REGION=$QA_REGION
else
    message UNKNOWN ENVIRONMENT
    echo 'You must specify an environment (bash deploy.sh <ENVIRONMENT>).'
    echo 'Allowed values are "staging", "qa", "private-prod" or "prod"'
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
docker push makerdao/vdb-headersync:$TAG
docker push makerdao/vdb-headersync:$IMMUTABLE_TAG

message PUSHING RESET-HEADER-CHECK
docker push makerdao/vdb-reset-header-check:$TAG
docker push makerdao/vdb-reset-header-check:$IMMUTABLE_TAG

# service deploy
message DEPLOYING HEADER-SYNC TO $ENVIRONMENT IN $REGION
aws ecs update-service --cluster vdb-cluster-$ENVIRONMENT --service vdb-header-sync-$ENVIRONMENT --force-new-deployment --endpoint https://ecs.$REGION.amazonaws.com --region $REGION

# announce deploy
.travis/announce.sh $ENVIRONMENT vdb-header-sync
