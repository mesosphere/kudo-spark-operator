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
  go test -v -count=1 -json -timeout 60m .
else
  go test -v -count=1 -timeout 60m .
fi

EXIT_CODE=$?

popd

exit $EXIT_CODE
