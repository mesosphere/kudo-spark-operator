#!/usr/bin/env bash

NAMESPACE=hdfs-kerberos

kubectl create namespace hdfs-kerberos
kubens ${NAMESPACE}

kubectl kudo init || true
kubectl kudo install spark --namespace=${NAMESPACE}

kubectl apply -f volumes
kubectl apply -f configmaps

kubectl apply -f kerberos-deployment.yml

kubectl apply -f namenode-deployment.yml
kubectl apply -f datanode-deployment.yml