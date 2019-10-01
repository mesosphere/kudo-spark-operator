#!/usr/bin/env bash

set -e

# The following environment variables are available to customize test execution:
# - KUBECONFIG
# - OPERATOR_IMAGE
# - SPARK_IMAGE
# - SPARK_VERSION
# Default values can be looked up in tests/utils/common.go
TEST_DIR=$(pwd) \
go test -v -count=1 -timeout 30m .
