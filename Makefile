.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared
SPARK_OPERATOR_DIR := $(ROOT_DIR)/spark-on-k8s-operator

KONVOY_VERSION ?= v1.1.5
export KONVOY_VERSION

CLUSTER_TYPE ?= konvoy

SPARK_ON_K8S_OPERATOR_REPO_NAME ?= mesosphere
SPARK_ON_K8S_OPERATOR_IMAGE_NAME ?= spark-on-k8s-operator
SPARK_ON_K8S_OPERATOR_FULL_IMAGE_NAME ?= $(SPARK_ON_K8S_OPERATOR_REPO_NAME)/$(SPARK_ON_K8S_OPERATOR_IMAGE_NAME)
SPARK_ON_K8S_DOCKER_FILE_PATH ?= shared/spark-on-k8s-operator
SPARK_DOCKER_IMAGE_NAME ?= "gcr.io/spark-operator/spark:v2.4.0"

.PHONY: cluster-create
cluster-create:
	if [[ ! -f  $(CLUSTER_TYPE)-created ]]; then
		$(KUDO_TOOLS_DIR)/cluster.sh $(CLUSTER_TYPE) up
		echo > $(CLUSTER_TYPE)-created
	fi

.PHONY: cluster-destroy
cluster-destroy:
	if [[ $(CLUSTER_TYPE) == konvoy ]]; then
		$(KUDO_TOOLS_DIR)/cluster.sh konvoy down
		rm -f konvoy-created
	else
		$(KUDO_TOOLS_DIR)/cluster.sh mke down
		rm -f mke-created
	fi

.PHONY: operator-docker-build
operator-docker-build:
	docker build \
		--build-arg SPARK_IMAGE=${SPARK_DOCKER_IMAGE_NAME} \
		-t ${SPARK_ON_K8S_OPERATOR_FULL_IMAGE_NAME} \
		${SPARK_ON_K8S_DOCKER_FILE_PATH}

.PHONY: operator-docker-push
operator-docker-push:
	docker push ${SPARK_ON_K8S_OPERATOR_FULL_IMAGE_NAME}

test:
	$(ROOT_DIR)/run-tests.sh

.PHONY: clean-all
clean-all:
	rm -f *.pem *.pub cluster.yaml cluster.tmp.yaml *-created
	rm -rf state runs

