#!/bin/bash
#
# This script submits a sorting application and requires Spark Operator to be installed.
# Dataset should be generated upfront.

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"

S3_ENDPOINT=${S3_ENDPOINT:-s3.us-west-2.amazonaws.com}
SERVICE_ACCOUNT_NAME=${SERVICE_ACCOUNT_NAME:-spark-service-account}

if [[ $# -lt 2 ]]; then
  echo "Usage:" >&2
  echo "  $0 <namespace> <S3 source path> <S3 target path>" >&2
  exit 1
fi

NAMESPACE=${1:-spark}
NUM_EXECUTORS=${NUM_EXECUTORS:-500}

. ${SCRIPT_DIR}/aws_credentials.sh

cat ${TEMPLATES_DIR}/sort-application.tmpl \
  | sed "s|AWS_ACCESS_KEY_ID|${AWS_ACCESS_KEY_ID:-}|" \
  | sed "s|AWS_SECRET_ACCESS_KEY|${AWS_SECRET_ACCESS_KEY:-}|" \
  | sed "s|AWS_SESSION_TOKEN|${AWS_SESSION_TOKEN:-}|" \
  | sed "s|S3_ENDPOINT|${S3_ENDPOINT}|" \
  | sed "s|SERVICE_ACCOUNT_NAME|${SERVICE_ACCOUNT_NAME}|" \
  | sed "s|NUM_EXECUTORS|${NUM_EXECUTORS}|" \
  | sed "s|SOURCE_PATH|${2:-}|" \
  | sed "s|TARGET_PATH|${3:-}|" \
  | kubectl --namespace "${NAMESPACE}" apply -f -
