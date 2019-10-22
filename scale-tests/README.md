Scale Tests Tooling
---

## Before you start

* `kubectl` should be configured and point to the cluster under test
* KUDO CLI initialized with `kubectl kudo init [--client-only]`

## Naming conventions

* namespace names have a format `spark-{N}`, where `N` is a sequential number
* Spark Operator instances names have format `spark-operator-{N}`, where `N` is a sequential number and usually is the same as namespace sequential number
* Spark Application names have a format `spark-test-ns-{N}-{M}`, where `N` is a namespace sequential number and `M` is application sequential number

## Scripts
Scripts rely on the naming conventions to deploy and remove the resources:
- [install.sh](scripts/install.sh) creates N namespaces and installs an operator instance in each
- [run.sh](scripts/run.sh) generates specs for M SparkApplication per namespace and submits them to N namespaces
- [uninstall.sh](scripts/uninstall.sh) deletes N namespaces

Example:
```bash
# Deploy CRDs, create 2 namespaces and create an operator instance in each
./scripts/install.sh 2

# Submit 2 applications to 2 operator instances
./scripts/run.sh 2 2

# Observe metrics and alerts and uninstall operators
# Delete operator instances, namespaces, and CRDs
./scripts/uninstall.sh 2
```

## Dashboards
Dashboards can be imported to Grafana from [dashboards](dashboards) folder.