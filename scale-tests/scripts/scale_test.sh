#!/bin/bash
#
# This script submits multiple noop Spark Applications exercising allocation and CPU usage.

set -ex
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
SCALE_TESTS_DIR="$(dirname "${SCRIPT_DIR}")"
PROJECT_ROOT_DIR="$(dirname "${SCALE_TESTS_DIR}")"

TEMPLATES_DIR="${SCALE_TESTS_DIR}/templates"

NAMESPACE_PREFIX=${NAMESPACE_PREFIX:-spark}
APP_NAME_PREFIX=${APP_NAME_PREFIX:-spark-mtr}
SERVICE_ACCOUNT_NAME=${SERVICE_ACCOUNT_NAME:-spark-service-account}

# To achieve even executors utilization and predictable application duration,
# NUM_EXECUTORS should be equal to NUM_TASKS. In this case end-to-end duration
# of the application will be TASK_DURATION_SEC
NUM_EXECUTORS=${NUM_EXECUTORS:-1}
NUM_TASKS=${NUM_TASKS:-1}
TASK_DURATION_SEC=${TASK_DURATION_SEC:-1800}


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
      SPARK_APP_NAME="${APP_NAME_PREFIX}-ns-${n}-${i}"

      APP_SPEC=$(cat ${TEMPLATES_DIR}/scale-test-application.tmpl \
        | sed "s|SPARK_APP_NAME|${SPARK_APP_NAME}|" \
        | sed "s|SERVICE_ACCOUNT_NAME|${SERVICE_ACCOUNT_NAME}|" \
        | sed "s|NUM_EXECUTORS|${NUM_EXECUTORS}|" \
        | sed "s|NUM_TASKS|${NUM_TASKS}|" \
        | sed "s|TASK_DURATION_SEC|${TASK_DURATION_SEC}|")

        MULTI_APP_SPEC=${MULTI_APP_SPEC}$'\n'"---"$'\n'"${APP_SPEC}"
    done
    echo "Submitting ${1} applications to namespace ${NAMESPACE}"
    echo "${MULTI_APP_SPEC}"
    echo "${MULTI_APP_SPEC}" | kubectl apply --namespace "${NAMESPACE}" -f -
done
