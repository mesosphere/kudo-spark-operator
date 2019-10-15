#!/usr/bin/env bash

NAMESPACE=${NAMESPACE:-default}

kubectl delete -n "${NAMESPACE}" operatorversion.kudo.dev \
    $(kubectl get -n "${NAMESPACE}" instances.kudo.dev -o jsonpath="{.items[?(@.metadata.labels.kudo\.dev/operator=='spark')].spec.operatorVersion.name}")
kubectl delete -n "${NAMESPACE}" instance.kudo.dev \
    $(kubectl get -n "${NAMESPACE}" instances.kudo.dev -o jsonpath="{.items[?(@.metadata.labels.kudo\.dev/operator=='spark')].metadata.name}")

# ToDo: Once these resources will be managed by the operator, remove these deletions from here
kubectl delete -n "${NAMESPACE}" service spark-application-metrics
kubectl delete -n "${NAMESPACE}" ServiceMonitor spark-cluster-monitor
kubectl delete crd sparkapplications.sparkoperator.k8s.io scheduledsparkapplications.sparkoperator.k8s.io
