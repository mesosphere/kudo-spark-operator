#!/usr/bin/env bash

set -e

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
pushd "${SCRIPTPATH}"/

echo "Building images"
make docker-push
echo "Running tests"
make test
echo "Tests finished"

popd
