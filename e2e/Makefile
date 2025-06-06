include ../Makefile.build.mk

BIN_DIR := $(shell pwd)/bin
KOTS_BIN_DIR := $(shell dirname $(shell pwd))/bin
PLAYWRIGHT_DIR := $(shell pwd)/playwright

SHELL := /bin/bash

OS ?= linux
ARCH ?= $(shell go env GOARCH)

.PHONY: all
all: build deps test

.PHONY: deps
deps:
	docker build --platform $(OS)/$(ARCH) -t e2e-deps .

.PHONY: build
build:
	GOOS=$(OS) GOARCH=$(ARCH) go test $(BUILDFLAGS) -c -o bin/e2e.test .

.PHONY: test
test: export GINKGO_EDITOR_INTEGRATION=1 # disable error on programatic focus
test: KOTSADM_IMAGE_TAG ?= alpha
test: AIRGAP ?= 0
test: IS_OPENSHIFT ?= 0
test: IS_EKS ?= 0
test: SKIP_TEARDOWN ?= 0
ifneq ($(EXISTING_KUBECONFIG),)
test: EXISTING_KUBECONFIG_VOLUME_MOUNT := "-v=$(EXISTING_KUBECONFIG):$(EXISTING_KUBECONFIG)"
endif
test:
	docker run --rm -i --net host \
		-v $(BIN_DIR)/e2e.test:/usr/local/bin/e2e.test \
		-e REPLICATED_API_TOKEN \
		-e GITOPS_GITHUB_TOKEN \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kots \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kubectl-kots \
		-v $(PLAYWRIGHT_DIR)/playwright-report:/playwright/playwright-report \
		$(EXISTING_KUBECONFIG_VOLUME_MOUNT) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		e2e-deps \
		e2e.test \
			-test.v \
			--ginkgo.v \
			--ginkgo.focus="$(FOCUS)" \
			--existing-kubeconfig=$(EXISTING_KUBECONFIG) \
			--kotsadm-image-registry=$(KOTSADM_IMAGE_REGISTRY) \
			--kotsadm-image-namespace=$(KOTSADM_IMAGE_NAMESPACE) \
			--kotsadm-image-tag=$(KOTSADM_IMAGE_TAG) \
			--airgap=$(AIRGAP) \
			--is-openshift=$(IS_OPENSHIFT) \
			--is-eks=$(IS_EKS) \
			--kots-helm-chart-url=$(KOTS_HELM_CHART_URL) \
			--kots-helm-chart-version=$(KOTS_HELM_CHART_VERSION) \
			--kots-dockerhub-username=$(KOTS_DOCKERHUB_USERNAME) \
			--kots-dockerhub-password=$(KOTS_DOCKERHUB_PASSWORD) \
			--aws-access-key-id=$(AWS_ACCESS_KEY_ID) \
			--aws-secret-access-key=$(AWS_SECRET_ACCESS_KEY) \
			--git-tag=$(GIT_TAG) \
			--skip-teardown=$(SKIP_TEARDOWN)

.PHONY: path
path:
	@echo "PATH=$(PATH)"
