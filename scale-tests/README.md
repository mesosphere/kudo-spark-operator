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
- [scale_test.sh](scripts/scale_test.sh) generates specs for M scale-test SparkApplications per namespace and submits them to N namespaces
- [terasort.sh](scripts/terasort.sh) runs a single TeraSort SparkApplication in a specified namespace
- [uninstall.sh](scripts/uninstall.sh) deletes N namespaces

Scale test example:
```bash
# Deploy CRDs, create 50 namespaces and create an operator instance in each
./scripts/install.sh 50

# Submit 20 applications to each of 50 operator instances (1000 total)
./scripts/scale_test.sh 20 50

# Observe metrics and alerts and uninstall operators
# Delete operator instances, namespaces, and CRDs
./scripts/uninstall.sh 50
```

TeraSort example:
```bash
# Deploy CRDs, create 1 namespaces and create an operator instance
./scripts/install.sh 1

# Submit TeraSort benchmark to the Operator instance
./scripts/terasort.sh spark-1 s3a://bucket/input s3a://bucket/output

# Observe metrics and alerts and uninstall operators
# Delete operator instances, namespaces, and CRDs
./scripts/uninstall.sh 1
```

## Dashboards
Dashboards can be imported to Grafana from [dashboards](dashboards) folder.