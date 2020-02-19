#!/usr/bin/env bash

set -x

NAMESPACE=hdfs-kerberos

kubectl create namespace hdfs-kerberos
kubens ${NAMESPACE}

kubectl apply -f volumes
kubectl apply -f configmaps

kubectl apply -f kerberos-deployment.yml
kubectl wait deployments/kerberos --for condition=available  --timeout=60s

kubectl apply -f namenode-deployment.yml
kubectl wait deployments/hdfs-namenode --for condition=available --timeout=60s

kubectl apply -f datanode-deployment.yml
kubectl wait deployments/hdfs-datanode --for condition=available --timeout=60s

# write test data to hdfs and export delegation token
namenode_pod_name=$(kubectl get pods --selector name=hdfs-namenode --output=jsonpath='{.items[*].metadata.name}')

kubectl exec "$namenode_pod_name" -- init.sh
kubectl cp "$namenode_pod_name":tmp/hadoop.token /tmp/hadoop.token
kubectl create secret generic hadoop-token --from-file /tmp/hadoop.token
