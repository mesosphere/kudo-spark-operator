#!/bin/bash

set -ex
SCRIPT_DIR=$(dirname "$0")
OPERATOR_DIR="$(dirname ${SCRIPT_DIR})/kudo-spark-operator/operator"

NAMESPACE=${NAMESPACE:-spark}
OPERATOR_DOCKER_REPO=${OPERATOR_DOCKER_REPO:-mesosphere/kudo-spark-operator}
OPERATOR_VERSION=${OPERATOR_VERSION:-3.0.0-1.1.0}

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

    kubectl kudo --namespace "${NAMESPACE}" install "${OPERATOR_DIR}" -p operatorImageName="${OPERATOR_DOCKER_REPO}" -p operatorVersion="${OPERATOR_VERSION}"
fi

