# KUDO Spark Operator

# Developing

### Prerequisites

Required software:
* Docker
* GNU Make 4.2.1 or higher
* sha1sum
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [KUDO CLI Plugin](https://kudo.dev/docs/#install-kudo-cli) 0.7.5 or higher

For test cluster provisioning and Stub Universe artifacts upload valid AWS access credentials required:
* `AWS_PROFILE` **or** `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables should be provided

For pulling private repos, a GitHub token is required:
* generate [GitHub token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) 
and export environment variable with token contents: `export GITHUB_TOKEN=<your token>`
  * or save the token either to `<repo root>/shared/data-services-kudo/.github_token` or to `~/.ds_kudo_github_token` 

### Build steps

GNU Make is used as the main build tool and includes the following main targets:
* `make cluster-create` creates a Konvoy or MKE cluster
* `make cluster-destroy` creates a Konvoy or MKE cluster
* `make clean-all` removes all artifacts produced by targets from local filesystem
* `make docker-spark` builds Spark base image based on Apache Spark 2.4.3
* `make docker-operator` builds Operator image and Spark base image if it's not built
* `make docker-builder` builds image with required tools to run tests
* `make docker-push` publishes Spark base image and Spark Operator image to DockerHub
* `make test` runs tests suite
* `make clean-docker` removes all files, created by `make` during `docker build` goals execution

A typical workflow looks as following:
```
make clean-all
make cluster-create
make docker-push 
make test
make cluster-destroy
```

To run tests on a pre-existing cluster with specified operator and spark images, set KUBECONFIG, SPARK_IMAGE_FULL_NAME and OPERATOR_IMAGE_FULL_NAME variables

```
make test KUBECONFIG=$HOME/.kube/config \
SPARK_IMAGE_FULL_NAME=mesosphere/spark:spark-2.4.3-hadoop-2.9-k8s \
OPERATOR_IMAGE_FULL_NAME=mesosphere/kudo-spark-operator:spark-2.4.3-hadoop-2.9-k8s
```

# Installing and using Spark Operator

### Prerequisites

* Kubernetes cluster up and running
* `kubectl` configured to work with provisioned cluster
* [KUDO CLI Plugin](https://kudo.dev/docs/#install-kudo-cli) 0.7.5 or higher

### Installation

To install KUDO Spark Operator, run:
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

To get started with your app monitoring, please, see also [monitoring documentation](kudo-operator/docs/latest/monitoring.md)

###  MKE cluster provisioning

If you want to create a cluster with MKE Kubernetes distribution, the following environment variables must be set before executing 
`make cluster-create` :

- DCOS_LICENSE - should be populated from a `licence.txt` file
- CLUSTER_TYPE - type of a cluster, in our case is `mke`
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- AWS_SESSION_TOKEN

AWS credentials are exported automatically by `make`, so there is no need to handle them manually, but `CLUSTER_TYPE` 
and `DCOS_LICENSE` need to be set manually.
```

$ maws li Team\ 10 #refresh AWS credentials
$ export CLUSTER_TYPE=mke
$ export DCOS_LICENSE=$(cat /path/to/the/license.txt)
$ make cluster-create
```

