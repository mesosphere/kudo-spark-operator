#! /bin/bash
printf "Running kinit..."
until kinit -kt /var/keytabs/hdfs.keytab hdfs/namenode.${NAMESPACE}.svc.cluster.local; do sleep 2; done

until (echo > /dev/tcp/namenode.${NAMESPACE}.svc.cluster.local/9000) >/dev/null 2>&1; do sleep 2; done

# add test data
hdfs dfsadmin -safemode wait

printf '\nCopying test data to HDFS...'
hdfs dfs -mkdir -p /spark
hdfs dfs -copyFromLocal /hadoop/README.txt /spark
hdfs dfs -ls /spark

# export delegation token
printf '\nExporting delegation token...'
hdfs fetchdt --renewer hdfs /var/keytabs/delegation.token
