.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
SCRIPTS_DIR := $(ROOT_DIR)/scripts
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared
SPARK_OPERATOR_DIR := $(ROOT_DIR)/spark-on-k8s-operator

KONVOY_VERSION ?= v1.1.5
export KONVOY_VERSION

MKE_CLUSTER_NAME=kubernetes-cluster1

NAMESPACE ?= spark-operator

CLUSTER_TYPE ?= konvoy
KUBECONFIG ?= $(ROOT_DIR)/admin.conf

DOCKER_REPO_NAME ?= mesosphere

SPARK_IMAGE_NAME ?= spark-2.4.4-bin-hadoop2.7-k8s
SPARK_IMAGE_DOCKERFILE_PATH ?= $(ROOT_DIR)/images/spark
SPARK_IMAGE_TAG ?= $(call get_sha1sum,$(SPARK_IMAGE_DOCKERFILE_PATH)/Dockerfile)
SPARK_IMAGE_FULL_NAME ?= $(DOCKER_REPO_NAME)/$(SPARK_IMAGE_NAME):$(SPARK_IMAGE_TAG)

OPERATOR_IMAGE_NAME ?= kudo-spark-operator
OPERATOR_DOCKERFILE_PATH ?= $(ROOT_DIR)/images/operator
OPERATOR_VERSION ?= $(call get_sha1sum,$(OPERATOR_DOCKERFILE_PATH)/Dockerfile)
OPERATOR_IMAGE_FULL_NAME ?= $(DOCKER_REPO_NAME)/$(OPERATOR_IMAGE_NAME):$(OPERATOR_VERSION)

DOCKER_BUILDER_IMAGE_NAME ?= spark-operator-docker-builder
DOCKER_BUILDER_DOCKERFILE_PATH ?= $(ROOT_DIR)/images/builder
DOCKER_BUILDER_IMAGE_TAG ?= $(call get_sha1sum,$(DOCKER_BUILDER_DOCKERFILE_PATH)/Dockerfile)
DOCKER_BUILDER_IMAGE_FULL_NAME ?= $(DOCKER_REPO_NAME)/$(DOCKER_BUILDER_IMAGE_NAME):$(DOCKER_BUILDER_IMAGE_TAG)

# define export variables so they're available in shell
export AWS_PROFILE ?=
export AWS_ACCESS_KEY_ID ?=
export AWS_SECRET_ACCESS_KEY ?=
export AWS_SESSION_TOKEN ?=

get_sha1sum = $(shell cat $1 | sha1sum | cut -d ' ' -f1)
get_aws_credential = $(shell grep $(AWS_PROFILE) -A 3 ~/.aws/credentials | tail -n3 | grep $1 | xargs | cut -d' ' -f3)

.PHONY: aws_credentials
aws_credentials:
	# if the variable is not set, set the value from credentials file
	$(eval AWS_ACCESS_KEY_ID := $(if $(AWS_ACCESS_KEY_ID),$(AWS_ACCESS_KEY_ID),$(call get_aws_credential,aws_access_key_id)))
	$(eval AWS_SECRET_ACCESS_KEY := $(if $(AWS_SECRET_ACCESS_KEY),$(AWS_SECRET_ACCESS_KEY),$(call get_aws_credential,aws_secret_access_key)))
	$(eval AWS_SESSION_TOKEN := $(if $(AWS_SESSION_TOKEN),$(AWS_SESSION_TOKEN),$(call get_aws_credential,aws_session_token)))

.PHONY: cluster-create
cluster-create: aws_credentials
cluster-create:
	if [[ ! -f  $(CLUSTER_TYPE)-created ]]; then
		$(KUDO_TOOLS_DIR)/cluster.sh $(CLUSTER_TYPE) up
		echo > $(CLUSTER_TYPE)-created
	fi

.PHONY: cluster-destroy
cluster-destroy: aws_credentials
cluster-destroy:
	if [[ $(CLUSTER_TYPE) == konvoy ]]; then
		$(KUDO_TOOLS_DIR)/cluster.sh konvoy down
		rm -f konvoy-created
	else
		$(KUDO_TOOLS_DIR)/cluster.sh mke down
		rm -f mke-created
		kubectl config unset users.$(MKE_CLUSTER_NAME)
		kubectl config delete-context $(MKE_CLUSTER_NAME)
		kubectl config delete-cluster $(MKE_CLUSTER_NAME)
	fi

spark-build:
	docker build \
		-t ${SPARK_IMAGE_FULL_NAME} \
		-f ${SPARK_IMAGE_DOCKERFILE_PATH}/Dockerfile \
		${SPARK_IMAGE_DOCKERFILE_PATH}
	echo "${SPARK_IMAGE_FULL_NAME}" > $@

operator-build: spark-build
	docker build \
		--build-arg SPARK_IMAGE=$(shell cat spark-build) \
		-t ${OPERATOR_IMAGE_FULL_NAME} \
		-f ${OPERATOR_DOCKERFILE_PATH}/Dockerfile \
		${SPARK_OPERATOR_DIR} && docker image prune -f --filter label=stage=spark-operator-builder

	echo "${OPERATOR_IMAGE_FULL_NAME}" > $@

.PHONY: docker-push
docker-push:
	docker push $(OPERATOR_IMAGE_FULL_NAME)

.PHONY: install
install:
	$(SCRIPTS_DIR)/install_operator.sh

docker-builder:
	docker build \
		-t $(DOCKER_BUILDER_IMAGE_FULL_NAME) \
		-f ${DOCKER_BUILDER_DOCKERFILE_PATH}/Dockerfile \
		${DOCKER_BUILDER_DOCKERFILE_PATH}
	echo $(DOCKER_BUILDER_IMAGE_FULL_NAME) > $@

.PHONY: test
test: docker-builder
test: operator-build
test:
	docker run -i --rm \
		-v $(ROOT_DIR)/tests:/tests \
		-v $(KUBECONFIG):/root/.kube/config \
		-e KUBECONFIG=/root/.kube/config \
		-e SPARK_IMAGE="$(shell cat $(ROOT_DIR)/spark-build)" \
		-e SPARK_OPERATOR_IMAGE="$(shell cat $(ROOT_DIR)/operator-build)" \
		$(shell cat $(ROOT_DIR)/docker-builder) \
		/bin/bash -c \
		"kubectl cluster-info && \
		echo \$$SPARK_IMAGE && echo \$$SPARK_OPERATOR_IMAGE"
		# tests entrypoint

.PHONY: clean-all
clean-all:
	rm -f *.pem *.pub cluster.yaml cluster.tmp.yaml *-created aws_credentials
	rm -rf state runs

.PHONY: clean-docker
clean-docker:
	rm -f *-build docker-builder

