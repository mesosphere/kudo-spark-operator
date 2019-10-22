#!/usr/bin/env bash

set -e

# The following environment variables are used to parameterize test execution:
# - KUBECONFIG
# - OPERATOR_IMAGE
# - SPARK_IMAGE
# - SPARK_VERSION
# Default values can be looked up in tests/utils/common.go
pushd $TEST_DIR

if [ -n "$TEAMCITY_VERSION" ]; then
  # Running in Teamcity, therefore we need to convert `go test` output accordingly
  go test -v -count=1 -timeout 30m . | go-test-teamcity
else
  go test -v -count=1 -timeout 30m .
fi


popd
