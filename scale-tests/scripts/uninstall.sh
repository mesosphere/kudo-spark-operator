#!/bin/bash

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

SPECS_DIR="${PROJECT_ROOT_DIR}/specs"

NAMESPACE_PREFIX=${NAMESPACE_PREFIX:-spark}
INSTANCE_NAME_PREFIX=${INSTANCE_NAME_PREFIX:-spark-operator}

if [[ $# -lt 1 ]]; then
  echo "Usage:" >&2
  echo "  $0 <number of operator instances (and namespaces) to remove>" >&2
  exit 1
fi

for i in $(seq ${1}); do
    NAMESPACE="${NAMESPACE_PREFIX}-${i}"
    kubectl delete --namespace "${NAMESPACE}" instance "${INSTANCE_NAME_PREFIX}-${i}"
    kubectl delete namespace "${NAMESPACE}"
done
