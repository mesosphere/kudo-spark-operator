#!/usr/bin/env bash
set -x

/usr/sbin/kdb5_util -P changeme create -s

/usr/sbin/kadmin.local -q "addprinc -randkey HTTP/server.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "ktadd -k /var/keytabs/server.keytab HTTP/server.${NAMESPACE}.svc.cluster.local"

/usr/sbin/kadmin.local -q "addprinc -randkey hdfs/namenode.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "addprinc -randkey HTTP/namenode.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "addprinc -randkey hdfs/datanode.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "addprinc -randkey HTTP/datanode.${NAMESPACE}.svc.cluster.local"

/usr/sbin/kadmin.local -q "ktadd -k /var/keytabs/hdfs.keytab hdfs/namenode.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "ktadd -k /var/keytabs/hdfs.keytab HTTP/namenode.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "ktadd -k /var/keytabs/hdfs.keytab hdfs/datanode.${NAMESPACE}.svc.cluster.local"
/usr/sbin/kadmin.local -q "ktadd -k /var/keytabs/hdfs.keytab HTTP/datanode.${NAMESPACE}.svc.cluster.local"

chown hdfs /var/keytabs/hdfs.keytab

keytool -genkey -alias namenode."${NAMESPACE}".svc.cluster.local -keyalg rsa -keysize 1024 -dname "CN=namenode.${NAMESPACE}.svc.cluster.local" -keypass changeme -keystore /var/keytabs/hdfs.jks -storepass changeme
keytool -genkey -alias datanode."${NAMESPACE}".svc.cluster.local -keyalg rsa -keysize 1024 -dname "CN=datanode.${NAMESPACE}.svc.cluster.local" -keypass changeme -keystore /var/keytabs/hdfs.jks -storepass changeme

chmod 700 /var/keytabs/hdfs.jks
chown hdfs /var/keytabs/hdfs.jks

krb5kdc -n -P /kdc_pid
