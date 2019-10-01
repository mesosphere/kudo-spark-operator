#!/usr/bin/env bash

# this is a helper script that could be used when creating K8s cluster via mke

export CLUSTER_TYPE=mke

if [[ -z "$DCOS_LICENSE" ]]; then
    echo "Error! 'DCOS_LICENSE' is not set."
    return 1
fi

# export AWS_* environment variables
echo 'Exporting AWS credentials...'

set -a
eval $(cat ~/.aws/credentials | awk -F ' = ' 'NF && NR>1 {printf("%s=%s\n", toupper($1), $2)}')
set +a

echo 'Done.'

