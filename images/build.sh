#!/bin/bash

SCRIPT_DIR=$(dirname "$0")
ROOT_DIR="$(dirname ${SCRIPT_DIR})"

pushd "${ROOT_DIR}"

make docker-push

popd
