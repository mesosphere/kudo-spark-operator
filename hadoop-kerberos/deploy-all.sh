#!/usr/bin/env bash

kubectl create namespace hdfs-kerberos

kubens hdfs-kerberos

kubectl apply -f volumes
kubectl apply -f configmaps
kubectl apply -f kerberos-deployment.yml
kubectl apply -f namenode-deployment.yml
kubectl apply -f datanode-deployment.yml