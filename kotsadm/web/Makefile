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
	
.PHONY: clean
clean:
	rm -rf node_modules
	rm -rf dist

.PHONY: serve
serve: deps
	node --max_old_space_size=6144 \
		./node_modules/webpack-dev-server/bin/webpack-dev-server.js \
		--config webpack.config.js \
		-w --debug --compress --color --env skaffold --mode development --hot \
		--host 0.0.0.0

.PHONY: build-local
build-local: webpack_build_local
build-local:
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env skaffold \
		--mode development \
		--compress

.PHONY: build-kotsadm
build-kotsadm: SHIP_CLUSTER_BUILD_VERSION = $(if ${GIT_TAG},${GIT_TAG},$(shell echo ${GIT_COMMIT} | cut -c1-7))
build-kotsadm:
	SHIP_CLUSTER_BUILD_VERSION=$(SHIP_CLUSTER_BUILD_VERSION) \
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env enterprise \
		--mode production

