include ../Makefile.build.mk

BIN_DIR := $(shell pwd)/bin
KOTS_BIN_DIR := $(shell dirname $(shell pwd))/bin

SHELL := /bin/bash

.PHONY: all
all: build deps test

.PHONY: deps
deps:
	docker build -t e2e-deps -f ./hack/Dockerfile ./hack

.PHONY: build
build:
	go test $(BUILDFLAGS) -c -o bin/e2e.test .

.PHONY: test
test: export GINKGO_EDITOR_INTEGRATION=1 # disable error on programatic focus
test: KOTSADM_IMAGE_TAG ?= alpha
test: TESTIM_BRANCH ?= master
test: AIRGAP ?= 0
test: IS_OPENSHIFT ?= 0
test: SKIP_TEARDOWN ?= 0
ifneq ($(EXISTING_KUBECONFIG),)
test: EXISTING_KUBECONFIG_VOLUME_MOUNT := "-v=$(EXISTING_KUBECONFIG):$(EXISTING_KUBECONFIG)"
endif
test:
	docker run --rm -i --net host \
		-e TESTIM_ACCESS_TOKEN \
		-v $(BIN_DIR)/e2e.test:/usr/local/bin/e2e.test \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kots \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kubectl-kots \
		$(EXISTING_KUBECONFIG_VOLUME_MOUNT) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		e2e-deps \
		e2e.test \
			-test.v \
			--ginkgo.v \
			--ginkgo.focus="$(FOCUS)" \
			--testim-branch=$(TESTIM_BRANCH) \
			--existing-kubeconfig=$(EXISTING_KUBECONFIG) \
			--kotsadm-image-registry=$(KOTSADM_IMAGE_REGISTRY) \
			--kotsadm-image-namespace=$(KOTSADM_IMAGE_NAMESPACE) \
			--kotsadm-image-tag=$(KOTSADM_IMAGE_TAG) \
			--airgap=$(AIRGAP) \
			--is-openshift=$(IS_OPENSHIFT) \
			--kots-helm-chart-url=$(KOTS_HELM_CHART_URL) \
			--kots-helm-chart-version=$(KOTS_HELM_CHART_VERSION) \
			--kots-dockerhub-username=$(KOTS_DOCKERHUB_USERNAME) \
			--kots-dockerhub-password=$(KOTS_DOCKERHUB_PASSWORD) \
			--skip-teardown=$(SKIP_TEARDOWN) 

.PHONY: path
path:
	@echo "PATH=$(PATH)"
