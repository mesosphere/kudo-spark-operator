#!/usr/bin/env bash

set -e

# The following environment variables are used to parameterize test execution:
# - KUBECONFIG
# - OPERATOR_IMAGE
# - SPARK_IMAGE
# - SPARK_VERSION
# Default values can be looked up in tests/utils/common.go
pushd $TEST_DIR
go test -v -count=1 -timeout 30m .
popd
