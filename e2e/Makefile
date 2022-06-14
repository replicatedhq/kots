include ../Makefile.build.mk

BIN_DIR := $(shell pwd)/bin
KOTS_BIN_DIR := $(shell dirname $(shell pwd))/bin

SHELL := /bin/bash

.PHONY: all
all: build deps test

.PHONY: deps
deps:
	docker build -t e2e-deps -f ./hack/Dockerfile .

.PHONY: build
build:
	go test $(BUILDFLAGS) -c -o bin/e2e.test .

.PHONY: test
test: export GINKGO_EDITOR_INTEGRATION=1 # disable error on programatic focus
test: KOTSADM_IMAGE_TAG ?= alpha
test: TESTIM_BRANCH ?= master
test: SKIP_TEARDOWN ?= 0
test:
	export TESTIM_ACCESS_TOKEN
	docker run --rm -it --net host \
		-e TESTIM_ACCESS_TOKEN \
		-v $(BIN_DIR)/e2e.test:/usr/local/bin/e2e.test \
		-v $(KOTS_BIN_DIR)/kots:/usr/local/bin/kots \
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
			--skip-teardown=$(SKIP_TEARDOWN)

.PHONY: path
path:
	@echo "PATH=$(PATH)"
