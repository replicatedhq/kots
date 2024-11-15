SHELL := /bin/bash

ARCH ?= $(shell go env GOARCH)
CURRENT_USER := $(if $(GITHUB_USER),$(GITHUB_USER),$(shell id -u -n))

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
MELANGE ?= $(LOCALBIN)/melange
APKO ?= $(LOCALBIN)/apko

## Version to use for building
VERSION ?= $(shell git describe --tags --match='[0-9]*.[0-9]*.[0-9]*')

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

image-tag = $(shell echo "$1" | sed 's/+/-/')

.PHONY: print-%
print-%:
	@echo -n $($*)

.PHONY: check-env-%
check-env-%:
	@ if [ "${${*}}" = "" ]; then \
		echo "Environment variable $* not set"; \
		exit 1; \
	fi

melange: $(MELANGE)
$(MELANGE): $(LOCALBIN)
	go install chainguard.dev/melange@latest && \
		test -s $(GOBIN)/melange && \
		ln -sf $(GOBIN)/melange $(LOCALBIN)/melange

apko: $(APKO)
$(APKO): $(LOCALBIN)
	go install chainguard.dev/apko@latest && \
		test -s $(GOBIN)/apko && \
		ln -sf $(GOBIN)/apko $(LOCALBIN)/apko

CHAINGUARD_TOOLS_USE_DOCKER = 0
ifeq ($(CHAINGUARD_TOOLS_USE_DOCKER),"1")
MELANGE_CACHE_DIR ?= /go/pkg/mod
APKO_CMD = docker run -v $(shell pwd):/work -w /work -v $(shell pwd)/build/.docker:/root/.docker cgr.dev/chainguard/apko
MELANGE_CMD = docker run --privileged --rm -v $(shell pwd):/work -w /work -v "$(shell go env GOMODCACHE)":${MELANGE_CACHE_DIR} cgr.dev/chainguard/melange
else
MELANGE_CACHE_DIR ?= cache/.melange-cache
APKO_CMD = apko
MELANGE_CMD = melange
endif

$(MELANGE_CACHE_DIR):
	mkdir -p $(MELANGE_CACHE_DIR)

.PHONY: apko-build
apko-build: ARCHS ?= $(ARCH)
apko-build: check-env-IMAGE apko-template
	cd build && ${APKO_CMD} \
		build apko.yaml ${IMAGE} apko.tar \
		--arch ${ARCHS}

.PHONY: apko-build-and-publish
apko-build-and-publish: ARCHS ?= $(ARCH)
apko-build-and-publish: check-env-IMAGE apko-template
	@bash -c 'set -o pipefail && cd build && ${APKO_CMD} publish apko.yaml ${IMAGE} --arch ${ARCHS} | tee digest'
	$(MAKE) apko-output-image

.PHONY: apko-login
apko-login:
	rm -f build/.docker/config.json
	@ { [ "${PASSWORD}" = "" ] || [ "${USERNAME}" = "" ] ; } || \
	${APKO_CMD} \
		login -u "${USERNAME}" \
		--password "${PASSWORD}" "${REGISTRY}"

.PHONY: apko-print-pkg-version
apko-print-pkg-version: ARCHS ?= $(ARCH)
apko-print-pkg-version: apko-template check-env-PACKAGE_NAME
		cd build && \
		${APKO_CMD} show-packages apko.yaml --arch=${ARCHS} | \
		grep ${PACKAGE_NAME} | \
		cut -s -d" " -f2 | \
		head -n1

.PHONY: apko-output-image
apko-output-image: check-env-IMAGE
	@digest=$$(cut -s -d'@' -f2 build/digest); \
	if [ -z "$$digest" ]; then \
		echo "error: no image digest found" >&2; \
		exit 1; \
	fi ; \
	echo "$(IMAGE)@$$digest" > build/image

.PHONY: melange-build
melange-build: ARCHS ?= $(ARCH)
melange-build: MELANGE_SOURCE_DIR ?= .
melange-build: $(MELANGE_CACHE_DIR) melange-template
	mkdir -p build
	${MELANGE_CMD} \
		keygen build/melange.rsa
	${MELANGE_CMD} \
		build build/melange.yaml \
		--arch ${ARCHS} \
		--signing-key build/melange.rsa \
		--cache-dir=$(MELANGE_CACHE_DIR) \
		--source-dir $(MELANGE_SOURCE_DIR) \
		--out-dir build/packages \
		--git-repo-url github.com/replicatedhq/kots


.PHONY: melange-template
melange-template: check-env-MELANGE_CONFIG check-env-GIT_TAG
	mkdir -p build
	envsubst '$${GIT_TAG}' < ${MELANGE_CONFIG} > build/melange.yaml

.PHONY: apko-template
apko-template: check-env-APKO_CONFIG check-env-GIT_TAG
	mkdir -p build
	envsubst '$${GIT_TAG}' < ${APKO_CONFIG} > build/apko.yaml
