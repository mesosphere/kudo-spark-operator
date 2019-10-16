#!/bin/bash

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"
SPECS_DIR="${PROJECT_ROOT_DIR}/specs"
OPERATOR_DIR="${PROJECT_ROOT_DIR}/kudo-operator"

NAMESPACE_PREFIX=${NAMESPACE_PREFIX:-spark}
INSTANCE_NAME_PREFIX=${INSTANCE_NAME_PREFIX:-spark-operator}
OPERATOR_VERSION=${OPERATOR_VERSION:-spark-2.4.3-hadoop2.9-k8s}

if [[ $# -lt 1 ]]; then
  echo "Usage:" >&2
  echo "  $0 <number of operator instances (and namespaces) to create>" >&2
  exit 1
fi

echo "Installing CRDs"
kubectl apply -f ${SPECS_DIR}/spark-applications-crds.yaml

for i in $(seq ${1}); do
    NAMESPACE="${NAMESPACE_PREFIX}-${i}"
    echo "Creating namespace $NAMESPACE"
    sed 's|SPARK_NAMESPACE|'"${NAMESPACE}"'|g' ${TEMPLATES_DIR}/namespace.tmpl | kubectl apply -f -

    kubectl kudo --namespace "${NAMESPACE}" install --instance "${INSTANCE_NAME_PREFIX}-${i}" "${OPERATOR_DIR}" -p operatorVersion="${OPERATOR_VERSION}"
    kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-driver-rbac.yaml
    # Metrics
    kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-operator-service.yaml
    kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-service-monitor.yaml
done
