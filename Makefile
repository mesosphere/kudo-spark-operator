.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
SCRIPTS_DIR := $(ROOT_DIR)/scripts
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared
SPARK_OPERATOR_DIR := $(ROOT_DIR)/spark-on-k8s-operator

KONVOY_VERSION ?= v1.1.5
export KONVOY_VERSION

NAMESPACE ?= spark
MKE_CLUSTER_NAME=kubernetes-cluster1

CLUSTER_TYPE ?= konvoy
KUBECONFIG ?= $(ROOT_DIR)/admin.conf

DOCKER_REPO_NAME ?= mesosphere

SPARK_IMAGE_NAME ?= spark
SPARK_IMAGE_DIR ?= $(ROOT_DIR)/images/spark
SPARK_IMAGE_TAG ?= $(call files_checksum,$(SPARK_IMAGE_DIR))
SPARK_IMAGE_FULL_NAME ?= $(DOCKER_REPO_NAME)/$(SPARK_IMAGE_NAME):$(SPARK_IMAGE_TAG)

OPERATOR_IMAGE_NAME ?= kudo-spark-operator
OPERATOR_IMAGE_DIR ?= $(ROOT_DIR)/images/operator
OPERATOR_VERSION ?= $(call files_checksum,$(SPARK_IMAGE_DIR) $(OPERATOR_IMAGE_DIR) $(SPARK_OPERATOR_DIR))
OPERATOR_IMAGE_FULL_NAME ?= $(DOCKER_REPO_NAME)/$(OPERATOR_IMAGE_NAME):$(OPERATOR_VERSION)

DOCKER_BUILDER_IMAGE_NAME ?= spark-operator-docker-builder
DOCKER_BUILDER_IMAGE_DIR ?= $(ROOT_DIR)/images/builder
DOCKER_BUILDER_IMAGE_TAG ?= $(call files_checksum,$(DOCKER_BUILDER_IMAGE_DIR))
DOCKER_BUILDER_IMAGE_FULL_NAME ?= $(DOCKER_REPO_NAME)/$(DOCKER_BUILDER_IMAGE_NAME):$(DOCKER_BUILDER_IMAGE_TAG)

# Cluster provisioining and teardown
export AWS_PROFILE ?=
export AWS_ACCESS_KEY_ID ?=
export AWS_SECRET_ACCESS_KEY ?=
export AWS_SESSION_TOKEN ?=

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

# Docker
docker-builder:
	if [[ -z "$(call local_image_exists,$(DOCKER_BUILDER_IMAGE_FULL_NAME))" ]]; then
		docker build \
			-t $(DOCKER_BUILDER_IMAGE_FULL_NAME) \
			-f ${DOCKER_BUILDER_IMAGE_DIR}/Dockerfile \
			${DOCKER_BUILDER_IMAGE_DIR}
	fi
	echo $(DOCKER_BUILDER_IMAGE_FULL_NAME) > $@

docker-spark:
	if [[ -z "$(call remote_image_exists,$(SPARK_IMAGE_NAME),$(SPARK_IMAGE_TAG))" ]]; then
		docker build \
			-t ${SPARK_IMAGE_FULL_NAME} \
			-f ${SPARK_IMAGE_DIR}/Dockerfile \
			${SPARK_IMAGE_DIR}
	fi
	echo "${SPARK_IMAGE_FULL_NAME}" > $@

docker-operator: docker-spark
docker-operator:
	if [[ -z "$(call remote_image_exists,$(OPERATOR_IMAGE_NAME),$(OPERATOR_VERSION))" ]]; then
		docker build \
			--build-arg SPARK_IMAGE=$(shell cat docker-spark) \
			-t ${OPERATOR_IMAGE_FULL_NAME} \
			-f ${OPERATOR_IMAGE_DIR}/Dockerfile \
			${SPARK_OPERATOR_DIR} && docker image prune -f --filter label=stage=spark-operator-builder
	fi
	echo "${OPERATOR_IMAGE_FULL_NAME}" > $@

docker-push: docker-spark
docker-push: docker-operator
docker-push:
	if [[ -z "$(call remote_image_exists,$(SPARK_IMAGE_NAME),$(SPARK_IMAGE_TAG))" ]]; then
		docker push $(SPARK_IMAGE_FULL_NAME)
	fi
	if [[ -z "$(call remote_image_exists,$(OPERATOR_IMAGE_NAME),$(OPERATOR_VERSION))" ]]; then
		docker push $(OPERATOR_IMAGE_FULL_NAME)
	fi

# Testing
.PHONY: test
test: docker-builder
test: docker-push
test:
	docker run -i --rm \
		-v $(ROOT_DIR)/tests:/tests \
		-v $(KUBECONFIG):/root/.kube/config \
		-e TEST_DIR=/tests \
		-e KUBECONFIG=/root/.kube/config \
		-e SPARK_IMAGE=$(SPARK_IMAGE_FULL_NAME) \
		-e OPERATOR_IMAGE=$(OPERATOR_IMAGE_FULL_NAME) \
		$(shell cat $(ROOT_DIR)/docker-builder) \
		/tests/run.sh

.PHONY: install
install:
	OPERATOR_IMAGE_NAME=$(DOCKER_REPO_NAME)/$(OPERATOR_IMAGE_NAME) OPERATOR_VERSION=$(OPERATOR_VERSION) $(SCRIPTS_DIR)/install_operator.sh

.PHONY: clean-docker
clean-docker:
	rm -f docker-*

.PHONY: clean-all
clean-all: clean-docker
clean-all:
	rm -f *.pem *.pub *-created aws_credentials
	rm -rf state runs .konvoy-* *checksum cluster.yaml cluster.tmp.yaml inventory.yaml admin.conf

# function for extracting the value of an AWS property passed as an argument
define get_aws_credential
$(shell grep $(AWS_PROFILE) -A 3 ~/.aws/credentials | tail -n3 | grep $1 | xargs | cut -d' ' -f3)
endef

# function for calculating global checksum of directories and files passed as arguments.
# to avoid inconsistencies with files ordering in Linux/Unix systems the final checksum is
# calculated based on sorted checksums of independent files stored in a temporary file
#
# arguments:
# $1 - space-separated list of directories and/or files
define files_checksum
$(shell find $1 -type f | xargs sha1sum | cut -d ' ' -f1 > tmp.checksum && sort tmp.checksum | sha1sum | cut -d' ' -f1)
endef

# arguments:
# $1 - image name without repo e.g. spark
# $2 - image tag e.g. latest
define remote_image_exists
$(shell curl --silent --fail --list-only --location https://index.docker.io/v1/repositories/$(DOCKER_REPO_NAME)/$1/tags/$2 2>/dev/null)
endef

define local_image_exists
$(shell docker images -q $1 2> /dev/null)
endef
