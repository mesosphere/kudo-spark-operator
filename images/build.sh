#!/bin/bash

set -e
SCRIPT_DIR=$(dirname "$0")
cd "${SCRIPT_DIR}/.."

make docker-push
