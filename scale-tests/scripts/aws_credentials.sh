#!/bin/bash

set -ex

if [[ -z "${AWS_PROFILE:-}" ]]; then
    echo "AWS_PROFILE is not provided"
    exit 1
fi

get_aws_credential() {
    credential_name=$1
    cat ~/.aws/credentials | grep ${AWS_PROFILE} -A3 | tail -n3 | grep $1 | xargs | cut -d' ' -f3
}

export AWS_ACCESS_KEY_ID="$(get_aws_credential aws_access_key_id)"
export AWS_SECRET_ACCESS_KEY="$(get_aws_credential aws_secret_access_key)"
export AWS_SESSION_TOKEN="$(get_aws_credential aws_session_token)"
