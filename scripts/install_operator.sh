#!/bin/bash

set -ex
SCRIPT_DIR=$(dirname "$0")
SPECS_DIR="$(dirname ${SCRIPT_DIR})/specs"
OPERATOR_DIR="$(dirname ${SCRIPT_DIR})/kudo-operator"

NAMESPACE=${NAMESPACE:-spark}
OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME:-mesosphere/kudo-spark-operator}
OPERATOR_VERSION=${OPERATOR_VERSION:-latest}

echo "Using namespace '${NAMESPACE}' for installation"

kubectl kudo init || true

if [[ $(kubectl kudo get instances | grep spark) ]]; then
    echo "Spark Operator with name already installed"
    echo "if you want to remove it run: remove_operator.py"
else
    cat <<EOF | kubectl apply -f -
    apiVersion: v1
    kind: Namespace
    metadata:
      name: "${NAMESPACE}"
      labels:
        name: "${NAMESPACE}"
EOF
    # For the time being we need to install CRDs manually because KUDO doesn't fully support it
    kubectl apply -f ${SPECS_DIR}/spark-applications-crds.yaml
    kubectl kudo --namespace "${NAMESPACE}" install "${OPERATOR_DIR}" -p operatorImageName="${OPERATOR_IMAGE_NAME}" -p operatorVersion="${OPERATOR_VERSION}"
fi

kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-driver-rbac.yaml

# Create ServiceMonitor (see prometheus-operator docs) for Spark.
# TODO: make the service-monitor.yaml part of kudo-spark-operator once the issue https://github.com/kudobuilder/kudo/issues/913 will be fixed.
kubectl apply --namespace "${NAMESPACE}" -f ${SPECS_DIR}/spark-service-monitor.yaml
