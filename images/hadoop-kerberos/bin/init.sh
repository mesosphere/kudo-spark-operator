#! /bin/bash
set -x

until kinit -kt /var/keytabs/hdfs.keytab hdfs/namenode."${NAMESPACE}".svc.cluster.local; do sleep 5; done
until (echo > /dev/tcp/namenode."${NAMESPACE}".svc.cluster.local/9000) >/dev/null 2>&1; do sleep 5; done

# add test data
hdfs dfsadmin -safemode wait

hdfs dfs -mkdir -p /spark /history
hdfs dfs -copyFromLocal /hadoop/README.txt /spark
hdfs dfs -ls /spark

# export delegation token
hdfs fetchdt --renewer hdfs /tmp/hadoop.token
