#!/bin/bash

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"

NAMESPACE_PREFIX=${NAMESPACE_PREFIX:-spark}
APP_NAME_PREFIX=${APP_NAME_PREFIX:-spark-sort}

if [[ $# -lt 2 ]]; then
  echo "Usage:" >&2
  echo "  $0 <number of applications to create> <number of namespaces to submit apps to>" >&2
  exit 1
fi

for n in $(seq ${2}); do
    NAMESPACE="${NAMESPACE_PREFIX}-${n}"
    echo "Generating applications spec for namespace ${NAMESPACE}"
    MULTI_APP_SPEC=""
    for i in $(seq ${1}); do
        APP_SPEC=$(sed 's|SPARK_APP_NAME|'"${APP_NAME_PREFIX}-ns-${n}-${i}"'|g' ${TEMPLATES_DIR}/application.tmpl)
        MULTI_APP_SPEC=${MULTI_APP_SPEC}$'\n'"---"$'\n'"${APP_SPEC}"
    done
    echo "Submitting ${1} applications to namespace ${NAMESPACE}"
    echo "${MULTI_APP_SPEC}" | kubectl apply --namespace "${NAMESPACE}" -f -
done
