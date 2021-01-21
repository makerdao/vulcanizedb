#! /usr/bin/env bash
# Announce a deploy

set -e

function message() {
    echo
    echo -----------------------------------
    echo "$@"
    echo -----------------------------------
    echo
}

if ! sentry-cli -V &> /dev/null
then
  curl -sL https://sentry.io/get-cli/ | bash
fi

ENVIRONMENT=$1
if [ "$ENVIRONMENT" == "prod" ]; then
TAG=latest
elif [ "$ENVIRONMENT" == "staging" ]; then
TAG=staging
else
   message UNKNOWN ENVIRONMENT
fi

PROCESS=$2
if [ -z "$ENVIRONMENT" ] || [-z "$PROCESS"]; then
    echo 'You must specifiy an environment and a process (bash deploy.sh <ENVIRONMENT> <PROCESS>).'
    echo 'Allowed values for environment are "staging" or "prod"'
    exit 1
fi

export SENTRY_ORG=makerdao-k0
export SENTRY_LOG_LEVEL=info
SENTRY_PROJECT=vulcanize
SENTRY_RELEASE=$PROCESS-$(sentry-cli releases propose-version)
sentry-cli releases new -p $SENTRY_PROJECT $SENTRY_RELEASE
sentry-cli releases finalize $SENTRY_RELEASE
sentry-cli releases deploys $SENTRY_RELEASE new -e $ENVIRONMENT
