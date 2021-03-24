.ONESHELL:
SHELL := /bin/bash
.SHELLFLAGS = -ec

ROOT_DIR := $(CURDIR)

NAMESPACE ?= spark-workloads

.PHONY: install
install:
	kubectl kudo install "${ROOT_DIR}"/../kudo-spark-operator/operator \
		--instance=spark \
		--namespace=$(NAMESPACE) \
		--wait \
		-p enableMetrics=true \
		-p sparkJobNamespace=$(NAMESPACE)
	kubectl apply -f $(ROOT_DIR)/app/mock-task-runner.yaml -n $(NAMESPACE)
	kubectl apply -f $(ROOT_DIR)/app/shuffle-app.yaml -n $(NAMESPACE)

.PHONY: uninstall
uninstall:
	kubectl delete -f $(ROOT_DIR)/app/mock-task-runner.yaml -n $(NAMESPACE)
	kubectl delete -f $(ROOT_DIR)/app/shuffle-app.yaml -n $(NAMESPACE)
	kubectl delete sparkapplications.sparkoperator.k8s.io --all -n $(NAMESPACE)
	kubectl kudo uninstall \
		--instance=spark \
        --namespace=$(NAMESPACE) \
        --wait
