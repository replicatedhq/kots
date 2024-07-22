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

.PHONY: serve
serve:
	node --max_old_space_size=6144 \
		./node_modules/webpack-dev-server/bin/webpack-dev-server.js \
		--config webpack.config.js \
		--color --env development

.PHONY: build-local
build-local:
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env development

.PHONY: build-kotsadm
build-kotsadm:
	rm -rf dist
	KOTSADM_BUILD_VERSION=$(GIT_TAG) \
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env enterprise

.PHONY: test-unit
test-unit:
	yarn test:unit

.PHONY: lint
lint:
	yarn prettier && yarn lint && yarn test:typecheck
