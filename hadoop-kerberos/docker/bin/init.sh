#!/usr/bin/env bash

echo "Resolving templates."

sed -i "s/{{NAMESPACE}}/${NAMESPACE}/g" /etc/krb5.conf
sed -i "s/{{NAMESPACE}}/${NAMESPACE}/g" "${HADOOP_CONF_DIR}"/*-site.xml