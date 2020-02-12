.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)
SCRIPTS_DIR := $(ROOT_DIR)/scripts
KUDO_TOOLS_DIR := $(ROOT_DIR)/shared
SPARK_OPERATOR_DIR := $(ROOT_DIR)/spark-on-k8s-operator

export KONVOY_VERSION ?= v1.1.5
export WORKER_NODE_INSTANCE_TYPE ?= m5.xlarge
export WORKER_NODE_COUNT ?= 5

export NAMESPACE ?= spark
MKE_CLUSTER_NAME=kubernetes-cluster1

CLUSTER_TYPE ?= konvoy
KUBECONFIG ?= $(ROOT_DIR)/admin.conf

SPARK_DOCKER_REPO ?= mesosphere/spark-dev
SPARK_IMAGE_DIR ?= $(ROOT_DIR)/images/spark
SPARK_IMAGE_TAG ?= $(call files_checksum,$(SPARK_IMAGE_DIR))
SPARK_IMAGE_FULL_NAME ?= $(SPARK_DOCKER_REPO):$(SPARK_IMAGE_TAG)

SPARK_RELEASE_DOCKER_REPO ?= mesosphere/spark

export OPERATOR_DOCKER_REPO ?= mesosphere/kudo-spark-operator-dev
export OPERATOR_VERSION ?= $(call files_checksum,$(SPARK_IMAGE_DIR) $(OPERATOR_IMAGE_DIR) $(SPARK_OPERATOR_DIR))
OPERATOR_IMAGE_DIR ?= $(ROOT_DIR)/images/operator
OPERATOR_IMAGE_FULL_NAME ?= $(OPERATOR_DOCKER_REPO):$(OPERATOR_VERSION)

OPERATOR_RELEASE_DOCKER_REPO ?= mesosphere/kudo-spark-operator

DOCKER_BUILDER_REPO ?= mesosphere/spark-operator-docker-builder
DOCKER_BUILDER_IMAGE_DIR ?= $(ROOT_DIR)/images/builder
DOCKER_BUILDER_IMAGE_TAG ?= $(call files_checksum,$(DOCKER_BUILDER_IMAGE_DIR))
DOCKER_BUILDER_IMAGE_FULL_NAME ?= $(DOCKER_BUILDER_REPO):$(DOCKER_BUILDER_IMAGE_TAG)

# Cluster provisioining and teardown
export AWS_PROFILE ?=
export AWS_ACCESS_KEY_ID ?=
export AWS_SECRET_ACCESS_KEY ?=
export AWS_SESSION_TOKEN ?=

AWS_BUCKET_NAME ?= "kudo-ds-ci-artifacts"
AWS_BUCKET_PATH ?= "kudo-spark-operator/tests"

.PHONY: aws_credentials
aws_credentials:
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
	if [[ -z "$(call remote_image_exists,$(SPARK_DOCKER_REPO),$(SPARK_IMAGE_TAG))" ]]; then
		docker build \
			-t ${SPARK_IMAGE_FULL_NAME} \
			-f ${SPARK_IMAGE_DIR}/Dockerfile \
			${SPARK_IMAGE_DIR}
	fi
	echo "${SPARK_IMAGE_FULL_NAME}" > $@

docker-operator: docker-spark
docker-operator:
	if [[ -z "$(call remote_image_exists,$(OPERATOR_DOCKER_REPO),$(OPERATOR_VERSION))" ]]; then
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
	if [[ -z "$(call remote_image_exists,$(SPARK_DOCKER_REPO),$(SPARK_IMAGE_TAG))" ]]; then
		docker push $(SPARK_IMAGE_FULL_NAME)
	fi
	if [[ -z "$(call remote_image_exists,$(OPERATOR_DOCKER_REPO),$(OPERATOR_VERSION))" ]]; then
		docker push $(OPERATOR_IMAGE_FULL_NAME)
	fi

# Testing
.PHONY: test
test: docker-builder
test: docker-push
test:
	docker run -i --rm \
		-v $(ROOT_DIR):/kudo-spark-operator \
		-v $(KUBECONFIG):/root/.kube/config \
		-e TEST_DIR=/kudo-spark-operator/tests \
		-e KUBECONFIG=/root/.kube/config \
		-e SPARK_IMAGE=$(SPARK_IMAGE_FULL_NAME) \
		-e OPERATOR_IMAGE=$(OPERATOR_IMAGE_FULL_NAME) \
		-e TEAMCITY_VERSION="$(TEAMCITY_VERSION)" \
		-e AWS_ACCESS_KEY_ID="$(AWS_ACCESS_KEY_ID)" \
		-e AWS_SECRET_ACCESS_KEY="$(AWS_SECRET_ACCESS_KEY)" \
		-e AWS_SESSION_TOKEN="$(AWS_SESSION_TOKEN)" \
		-e AWS_BUCKET_NAME="$(AWS_BUCKET_NAME)" \
		-e AWS_BUCKET_PATH="$(AWS_BUCKET_PATH)" \
		$(shell cat $(ROOT_DIR)/docker-builder) \
		/kudo-spark-operator/tests/run.sh

.PHONY: install
install:
	OPERATOR_DOCKER_REPO=$(OPERATOR_DOCKER_REPO) OPERATOR_VERSION=$(OPERATOR_VERSION) $(SCRIPTS_DIR)/install_operator.sh

.PHONY: release-spark
release-spark: docker-spark
release-spark:
	$(call tag_and_push_image,$(SPARK_RELEASE_DOCKER_REPO),$(SPARK_IMAGE_RELEASE_TAG),$(SPARK_IMAGE_FULL_NAME))

.PHONY: release-operator
release-operator: docker-operator
release-operator:
	$(call tag_and_push_image,$(OPERATOR_RELEASE_DOCKER_REPO),$(OPERATOR_IMAGE_RELEASE_TAG),$(OPERATOR_IMAGE_FULL_NAME))

.PHONY: clean-docker
clean-docker:
	rm -f docker-*

.PHONY: clean-all
clean-all: clean-docker
clean-all:
	rm -f *.pem *.pub *-created aws_credentials
	rm -rf state runs .konvoy-* *checksum cluster.*yaml* inventory.yaml admin.conf

# function for extracting the value of an AWS property passed as an argument
define get_aws_credential
$(if $(AWS_PROFILE),$(shell cat ~/.aws/credentials | grep ${AWS_PROFILE} -A3 | tail -n3 | grep $1 | xargs | cut -d' ' -f3),$(error AWS_PROFILE is not set))
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
# $1 - image name with repo e.g. mesosphere/spark
# $2 - image tag e.g. latest
define remote_image_exists
$(shell curl --silent --fail --list-only --location https://index.docker.io/v1/repositories/$1/tags/$2 2>/dev/null)
endef

define local_image_exists
$(shell docker images -q $1 2> /dev/null)
endef

# arguments:
# $1 - release image repo, e.g. mesosphere/spark
# $2 - release image tag, e.g spark-2.4.3-hadoop-2.9-k8s
# $3 - dev image full name, e.g mesosphere/spark-dev:ab36f1f3691a8be2050f3acb559c34e3e8e5d66e
define tag_and_push_image
	$(eval RELEASE_IMAGE_FULL_NAME=$(1):$(2))
	# check, if specified image already exists to prevent overwrites
	if [[ -z "$(call remote_image_exists,$(1),$(2))" ]]; then
		docker pull $(3)
		docker tag $(3) $(RELEASE_IMAGE_FULL_NAME)
		echo "Pushing image \"$(RELEASE_IMAGE_FULL_NAME)\""
		docker push $(RELEASE_IMAGE_FULL_NAME)
	else
		echo "Error: image \"$(RELEASE_IMAGE_FULL_NAME)\" already exists, will not proceed with overwrite."; false
	fi
endef
