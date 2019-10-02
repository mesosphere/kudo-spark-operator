#!/bin/bash

set -ex
SCRIPT_DIR=$(dirname "$0")
SPECS_DIR="$(dirname ${SCRIPT_DIR})/specs"
OPERATOR_DIR="$(dirname ${SCRIPT_DIR})/kudo-operator"

NAMESPACE=${NAMESPACE:-spark}
NAMESPACE=${NAMESPACE:-spark-operator}

if [ -f "${SCRIPT_DIR}"/../operator-build ]; then
  echo "operator-build file is found."
  if [ -z "${OPERATOR_IMAGE_NAME}" ]; then
    echo "Parse OPERATOR_IMAGE_NAME from operator-build file."
    OPERATOR_IMAGE_NAME=$(awk -F':' '{printf $1}' <"${SCRIPT_DIR}"/../operator-build)
  fi
  if [ -z "${OPERATOR_VERSION}" ]; then
     echo "Parse OPERATOR_VERSION from operator-build file."
     OPERATOR_VERSION=$(awk -F':' '{printf $2}' <"${SCRIPT_DIR}"/../operator-build)
  fi
fi

OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME:-mesosphere/kudo-spark-operator}
OPERATOR_VERSION=${OPERATOR_VERSION:-latest}

echo "Using namespace '${NAMESPACE}' for installation"

kubectl kudo init || true

if [[ $(kubectl kudo get instances | grep spark) ]]; then
    echo "Spark Operator with name already installed"
    echo "if you want to remove it run: remove_operator.py"
else
    kubectl apply -f ${SPECS_DIR}/spark-namespace.yaml
    kubectl kudo --namespace "${NAMESPACE}" install ${OPERATOR_DIR} -p operatorImageName="${OPERATOR_IMAGE_NAME}" -p operatorVersion="${OPERATOR_VERSION}"
fi

kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-driver-rbac.yaml
# Expose Spark Operator metrics service
kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-operator-service.yaml
# Create ServiceMonitor (see prometheus-operator docs) for Spark
kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-service-monitor.yaml
