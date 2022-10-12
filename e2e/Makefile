include ../Makefile.build.mk

SCRIPT_MOUNT_SRC := $(shell dirname $(shell pwd))/bin$(SCRIPT_NAME)
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
test: SKIP_TEARDOWN ?= 0
ifneq ($(EXISTING_KUBECONFIG),)
test: EXISTING_KUBECONFIG_VOLUME_MOUNT := "-v=$(EXISTING_KUBECONFIG):$(EXISTING_KUBECONFIG)"
endif
ifneq ($(SCRIPT_NAME),)
test: RUN_SCRIPT_VOLUME_MOUNT := "-v=$(SCRIPT_MOUNT_SRC):$(SCRIPT_NAME)"
endif
test:
	docker run --rm -i --net host \
		-e TESTIM_ACCESS_TOKEN \
		-v $(BIN_DIR)/e2e.test:/usr/local/bin/e2e.test \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kots \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kubectl-kots \
		$(EXISTING_KUBECONFIG_VOLUME_MOUNT) \
		$(RUN_SCRIPT_VOLUME_MOUNT) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		e2e-deps \
		e2e.test \
			-test.v \
			--ginkgo.focus="$(FOCUS)" \
			--testim-branch=$(TESTIM_BRANCH) \
			--existing-kubeconfig=$(EXISTING_KUBECONFIG) \
			--kotsadm-image-registry=$(KOTSADM_IMAGE_REGISTRY) \
			--kotsadm-image-namespace=$(KOTSADM_IMAGE_NAMESPACE) \
			--kotsadm-image-tag=$(KOTSADM_IMAGE_TAG) \
			--kots-helm-chart-url=$(KOTS_HELM_CHART_URL) \
			--kots-helm-chart-version=$(KOTS_HELM_CHART_VERSION) \
			--script-name=$(SCRIPT_NAME) \
			--script-is-async=$(SCRIPT_IS_ASYNC) \
			--script-check-url=$(SCRIPT_CHECK_URL) \
			--script-check-status-code=$(SCRIPT_HTTP_CHECK_STATUS_CODE) \
			--skip-teardown=$(SKIP_TEARDOWN)

.PHONY: path
path:
	@echo "PATH=$(PATH)"
