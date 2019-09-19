.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared/data-services-kudo
SPARK_OPERATOR_DIR := $(ROOT_DIR)/shared/spark-on-k8s-operator

KONVOY_VERSION ?= v1.1.5
export KONVOY_VERSION

CLUSTER_TYPE ?= konvoy

cluster-create:
	$(KUDO_TOOLS_DIR)/cluster.sh $(CLUSTER_TYPE) up
	echo > $(CLUSTER_TYPE)-created

cluster-destroy:
	if [[ -f konvoy-created ]]; then
		$(KUDO_TOOLS_DIR)/cluster.sh konvoy down
	fi
	if [[ -f mke-created ]]; then
		$(KUDO_TOOLS_DIR)/cluster.sh mke down
	fi

.PHONY: clean-all
clean-all:
	rm -f *.pem *.pub cluster.yaml cluster.tmp.yaml
