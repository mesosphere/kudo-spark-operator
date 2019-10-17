#!/bin/bash

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"

NAMESPACE_PREFIX=${NAMESPACE_PREFIX:-spark}
APP_NAME_PREFIX=${APP_NAME_PREFIX:-spark-sort}

S3_PATH=${S3_PATH:-spark-sort}
NUM_EXECUTORS=${NUM_EXECUTORS:-4}

if [[ $# -lt 2 ]]; then
  echo "Usage:" >&2
  echo "  $0 <number of applications to create> <number of namespaces to submit apps to>" >&2
  exit 1
fi

. ${SCRIPT_DIR}/aws_credentials.sh

for n in $(seq ${2}); do
    NAMESPACE="${NAMESPACE_PREFIX}-${n}"
    echo "Generating applications spec for namespace ${NAMESPACE}"
    MULTI_APP_SPEC=""
    for i in $(seq ${1}); do
      SPARK_APP_NAME="${APP_NAME_PREFIX}-ns-${n}-${i}"

      APP_SPEC=$(cat ${TEMPLATES_DIR}/sort-application.tmpl \
        | sed "s|AWS_ACCESS_KEY_ID|${AWS_ACCESS_KEY_ID:-}|" \
        | sed "s|AWS_SECRET_ACCESS_KEY|${AWS_SECRET_ACCESS_KEY:-}|" \
        | sed "s|AWS_SESSION_TOKEN|${AWS_SESSION_TOKEN:-}|" \
        | sed "s|S3_ENDPOINT|${S3_ENDPOINT}|" \
        | sed "s|SPARK_APP_NAME|${SPARK_APP_NAME}|" \
        | sed "s|NUM_EXECUTORS|${NUM_EXECUTORS}|" \
        | sed "s|S3_PATH|${S3_PATH}|")

        MULTI_APP_SPEC=${MULTI_APP_SPEC}$'\n'"---"$'\n'"${APP_SPEC}"
    done
    echo "Submitting ${1} applications to namespace ${NAMESPACE}"
    echo "${MULTI_APP_SPEC}"
    echo "${MULTI_APP_SPEC}" | kubectl apply --namespace "${NAMESPACE}" -f -
done
