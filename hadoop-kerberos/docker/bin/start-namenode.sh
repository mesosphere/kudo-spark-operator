#!/usr/bin/env bash
set -x

until kinit -kt /var/keytabs/hdfs.keytab hdfs/namenode."${NAMESPACE}".svc.cluster.local; do sleep 15; done

echo "KDC is up and ready to go... starting up"

kdestroy

hdfs namenode -format
hdfs namenode
