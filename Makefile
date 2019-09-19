.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared/data-services-kudo
SPARK_OPERATOR_DIR := $(ROOT_DIR)/shared/spark-on-k8s-operator

KONVOY_VERSION ?= v1.1.5
export KONVOY_VERSION

cluster-create-konvoy:
	$(KUDO_TOOLS_DIR)/cluster.sh konvoy up

cluster-destroy-konvoy:
	$(KUDO_TOOLS_DIR)/cluster.sh konvoy down

.PHONY: cluster-destroy-all
destroy-all: cluster-destroy-konvoy

.PHONY: clean-all
clean-all:
	rm -f *.pem *.pub cluster.yaml cluster.tmp.yaml