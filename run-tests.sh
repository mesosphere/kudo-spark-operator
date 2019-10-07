#!/usr/bin/env bash

set -e

SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
pushd "${SCRIPTPATH}"/

make test

popd
