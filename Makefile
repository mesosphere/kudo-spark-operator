.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared
SPARK_OPERATOR_DIR := $(ROOT_DIR)/spark-on-k8s-operator

KONVOY_VERSION ?= v1.1.5
export KONVOY_VERSION

CLUSTER_TYPE ?= konvoy

SPARK_IMAGE_NAME ?= mesosphere/spark-2.4.4-bin-hadoop2.7-k8s
SPARK_ON_K8S_OPERATOR_IMAGE_NAME ?= mesosphere/spark-on-k8s-operator
SPARK_ON_K8S_DOCKER_FILE_PATH ?= images/operator/Dockerfile

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

.PHONY: spark-build
spark-build:
	docker build -t ${SPARK_IMAGE_NAME} images/spark

.PHONY: docker-build
docker-build:
	docker build -t ${SPARK_ON_K8S_OPERATOR_IMAGE_NAME} \
	-f ${SPARK_ON_K8S_DOCKER_FILE_PATH} shared/spark-on-k8s-operator

.PHONY: docker-push
docker-push:
	docker push ${SPARK_ON_K8S_OPERATOR_FULL_IMAGE_NAME}

test:
	$(ROOT_DIR)/run-tests.sh

.PHONY: clean-all
clean-all:
	rm -f *.pem *.pub cluster.yaml cluster.tmp.yaml *-created
	rm -rf state runs

