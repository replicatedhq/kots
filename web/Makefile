SHELL := /bin/bash

#paths within WSL start with /mnt/c/...
#docker does not recognize this fact
#this strips the first 5 characters (leaving /c/...) if the kernel releaser is Microsoft
ifeq ($(shell uname -r | tail -c 10), Microsoft)
	BUILD_DIR := $(shell pwd | cut -c 5-)
else
	BUILD_DIR := $(shell pwd)
endif

.PHONY: deps
deps:
	yarn --silent --frozen-lockfile

.PHONY: publish-pact
publish-pact:
	yarn run publish:pact

.PHONY: clean
clean:
	rm -rf node_modules
	rm -rf dist

.PHONY: test
test: deps
	rm -rf pacts/kotsadm-web-kotsadm-api.json
	yarn test

.PHONY: verify_local
verify_local:
	yarn test
	mkdir -p ../ship-cloud-api/pacts
	cp pacts/* ../ship-cloud-api/pacts

.PHONY: serve
serve: deps
	node --max_old_space_size=6144 \
		./node_modules/webpack-dev-server/bin/webpack-dev-server.js \
		--config webpack.config.js \
		-w --debug --color --env development

.PHONY: build-local
build-local: deps
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env development

.PHONY: build-kotsadm
build-kotsadm: KOTSADM_BUILD_VERSION = $(if ${GIT_TAG},${GIT_TAG},$(shell echo ${GIT_COMMIT} | cut -c1-7))
build-kotsadm: deps
	KOTSADM_BUILD_VERSION=$(KOTSADM_BUILD_VERSION) \
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env enterprise

