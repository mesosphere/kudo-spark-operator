#!/bin/bash

set -ex
SCRIPT_DIR=$(dirname "$0")
SPECS_DIR="$(dirname ${SCRIPT_DIR})/specs"

NAMESPACE=${NAMESPACE:-spark-operator}

OPERATOR_NAME=${OPERATOR_NAME:-spark-operator}
OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME:-mesosphere/kudo-spark-operator}
OPERATOR_VERSION=${OPERATOR_VERSION:-latest}

echo "Using namespace '${NAMESPACE}' for installation"

helm repo add incubator http://storage.googleapis.com/kubernetes-charts-incubator

if [[ $(helm ls | grep "${OPERATOR_NAME}") ]]; then
    echo "Spark Operator with name '${OPERATOR_NAME}' already installed"
    echo "if you want to remove it run: helm delete --purge ${OPERATOR_NAME}"
else
    helm install incubator/sparkoperator --namespace "${NAMESPACE}" --name "${OPERATOR_NAME}" \
    --set enableWebhook=true,sparkJobNamespace="${NAMESPACE}",enableMetrics=true,operatorImageName="${OPERATOR_IMAGE_NAME}",operatorVersion="${OPERATOR_VERSION}"
fi

kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-driver-rbac.yaml
kubeclt apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-events-pvc.yaml
kubeclt apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-history-server.yaml
kubeclt apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-historyserver-service.yaml
