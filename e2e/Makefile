include ../Makefile.build.mk

BIN_DIR := $(shell pwd)/bin

SHELL := /bin/bash
PATH := $(BIN_DIR):$(shell npm bin --prefix=$(BIN_DIR)):$(shell dirname $(shell pwd))/bin:$(PATH)

.PHONY: all
all: build test

.PHONY: deps
deps: export INSTALL_DIR=$(BIN_DIR)
deps:
	./hack/deps.sh

.PHONY: build
build:
	go test $(BUILDFLAGS) -c -o bin/e2e.test .

.PHONY: test
test: export GINKGO_EDITOR_INTEGRATION=1 # disable error on programatic focus
test: KOTSADM_IMAGE_TAG ?= v0.0.0-nightly
test: TESTIM_BRANCH ?= master
test: SKIP_TEARDOWN ?= 0
test:
	e2e.test \
		-test.v \
		--ginkgo.focus="$(FOCUS)" \
		--testim-branch=$(TESTIM_BRANCH) \
		--existing-kubeconfig=$(EXISTING_KUBECONFIG) \
		--kotsadm-image-registry=$(KOTSADM_IMAGE_REGISTRY) \
		--kotsadm-image-namespace=$(KOTSADM_IMAGE_NAMESPACE) \
		--kotsadm-image-tag=$(KOTSADM_IMAGE_TAG) \
		--skip-teardown=$(SKIP_TEARDOWN)

.PHONY: path
path:
	@echo "PATH=$(PATH)"
