# KUDO Spark Operator

# Developing

### Prerequisites

Required software:
* Docker
* GNU Make 4.2.1 or higher
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

For test cluster provisioning and Stub Universe artifacts upload valid AWS access credentials required:
* `AWS_PROFILE` **or** `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables should be provided

For pulling private repos, a GitHub token is required:
* generate [GitHub token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) 
and export environment variable with token contents: `export GITHUB_TOKEN=<your token>`
  * or save the token either to `<repo root>/shared/data-services-kudo/.github_token` or to `~/.ds_kudo_github_token` 

### Build steps

GNU Make is used as the main build tool and includes the following main targets:
* `make cluster-create-[konvoy|mke]` creates a Konvoy or MKE cluster
* `make cluster-destroy-[konvoy|mke]` creates a Konvoy or MKE cluster
* `make cluster-destroy-all` destroys all clusters created by `make cluster-create-[konvoy|mke]`
* `make clean-all` removes all artifacts produced by targets from local filesystem
* `make docker-build` builds all the images: Spark Base image and Spark Operator image 
* `make spark-build` builds Spark base image based on Apache Spark 2.4.4
* `make docker-push` publishes Spark Operator image to DockerHub

A typical workflow looks as following:
```
make clean-all
make cluster-create
make test
make cluster-destroy
```

# Installing and using Spark Operator

### Prerequisites

* Kubernetes cluster up and running
* `kubectl` configured to work with provisioned cluster
* `helm` client

### Installation

To install Spark Operator from Helm Chart, run:
```bash
make install
```

This make target runs [install_operator.sh](scripts/install_operator.sh) script which will install Spark Operator and 
create Spark Driver roles defined in [specs/spark-driver-rbac.yaml](specs/spark-driver-rbac.yaml). By default, Operator 
and Driver roles will be created and configured to run in namespace `spark-operator`. To change the namespace, 
provide `NAMESPACE` parameter to make:
```bash
make install NAMESPACE=test-namespace
```

### Submitting Spark Application

To submit Spark Application and check its status run:
```bash
#switch to operator namespace, e.g.
kubens spark-operator

# create Spark application
kubectl create -f specs/spark-application.yaml

# list applications
kubectl get sparkapplication

# check application status
kubectl describe sparkapplication mock-task-runner
```