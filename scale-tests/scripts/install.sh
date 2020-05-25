#!/bin/bash

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"
SPECS_DIR="${PROJECT_ROOT_DIR}/specs"
OPERATOR_DIR="${PROJECT_ROOT_DIR}/operators/repository/spark/operator"
SERVICE_ACCOUNT_NAME=${SERVICE_ACCOUNT_NAME:-spark-service-account}

NAMESPACE_PREFIX=${NAMESPACE_PREFIX:-spark}
INSTANCE_NAME_PREFIX=${INSTANCE_NAME_PREFIX:-spark-operator}
OPERATOR_VERSION=${OPERATOR_VERSION:-2.4.5-1.0.0}

if [[ $# -lt 1 ]]; then
  echo "Usage:" >&2
  echo "  $0 <number of operator instances (and namespaces) to create>" >&2
  exit 1
fi

for i in $(seq ${1}); do
    NAMESPACE="${NAMESPACE_PREFIX}-${i}"
    echo "Creating namespace $NAMESPACE"
    sed 's|SPARK_NAMESPACE|'"${NAMESPACE}"'|g' ${TEMPLATES_DIR}/namespace.tmpl | kubectl apply -f -
    sed 's|SERVICE_ACCOUNT_NAME|'"${SERVICE_ACCOUNT_NAME}"'|g' ${TEMPLATES_DIR}/service-account.tmpl | kubectl apply --namespace "${NAMESPACE}" -f -

    kubectl kudo --namespace "${NAMESPACE}" install --instance "${INSTANCE_NAME_PREFIX}-${i}" "${OPERATOR_DIR}" \
            -p operatorVersion="${OPERATOR_VERSION}" \
            -p sparkServiceAccountName="${SERVICE_ACCOUNT_NAME}" \
            -p createSparkServiceAccount=false \
            -p sparkJobNamespace="${NAMEPSACE}"

    kubectl wait  --for=condition=Available deployment --all  --namespace "$NAMESPACE" --timeout=120s
done
