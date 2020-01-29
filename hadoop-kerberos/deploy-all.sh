#!/usr/bin/env bash

k create namespace hdfs-kerberos

kubens hdfs-kerberos

k apply -f volumes
k apply -f configmaps
k apply -f kerberos-deployment.yml
k apply -f namenode-deployment.yml
k apply -f datanode-deployment.yml