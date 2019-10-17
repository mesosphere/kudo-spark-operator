#!/bin/bash
#
# This script runs a dataset generator for sorting benchmark and requires Spark Operator to be installed.
# Dataset should be generated upfront, this script is provided for convenience.

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"

S3_ENDPOINT=${S3_ENDPOINT:-s3.us-west-2.amazonaws.com}

if [[ $# -lt 2 ]]; then
  echo "Usage:" >&2
  echo "  $0 <namespace> <target s3 path>" >&2
  exit 1
fi

NAMESPACE=${1:-spark}
NUM_EXECUTORS=${NUM_EXECUTORS:-100}

. ${SCRIPT_DIR}/aws_credentials.sh

cat ${TEMPLATES_DIR}/gensort-application.tmpl \
  | sed "s|AWS_ACCESS_KEY_ID|${AWS_ACCESS_KEY_ID:-}|" \
  | sed "s|AWS_SECRET_ACCESS_KEY|${AWS_SECRET_ACCESS_KEY:-}|" \
  | sed "s|AWS_SESSION_TOKEN|${AWS_SESSION_TOKEN:-}|" \
  | sed "s|S3_ENDPOINT|${S3_ENDPOINT}|" \
  | sed "s|NUM_EXECUTORS|${NUM_EXECUTORS}|" \
  | sed "s|TARGET_S3_PATH|${2:-}|" | kubectl --namespace "${NAMESPACE}" apply -f -
