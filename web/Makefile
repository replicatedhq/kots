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
		--progress -w --debug --compress --color --env skaffold --mode development --hot \
		--host 0.0.0.0

.PHONY: build-kotsadm
build-kotsadm: SHIP_CLOUD_ENV = enterprise
build-kotsadm: webpack_build

.PHONY: build-local
build-local: SHIP_CLOUD_ENV = skaffold
build-local: webpack_build_local

webpack_build_local:
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env ${SHIP_CLOUD_ENV} \
		--mode development \
		--compress

.PHONY: webpack_build
webpack_build: SHIP_CLUSTER_BUILD_VERSION = $(if ${GIT_TAG},${GIT_TAG},$(shell echo ${BUILDKITE_COMMIT} | cut -c1-7))
webpack_build:
	SHIP_CLUSTER_BUILD_VERSION=$(SHIP_CLUSTER_BUILD_VERSION) \
	node \
		--max_old_space_size=6144 \
		./node_modules/webpack/bin/webpack.js \
		--config webpack.config.js \
		--env ${SHIP_CLOUD_ENV} \
		--mode production

.PHONY: publish-kotsadm-tag
publish-kotsadm-tag:
	docker build --build-arg=nginxconf=deploy/kotsadm.conf -f deploy/Dockerfile -t kotsadm/kotsadm-web:${GIT_TAG} .
	docker push kotsadm/kotsadm-web:${GIT_TAG}

.PHONY: publish-kotsadm-alpha
publish-kotsadm-alpha:
	docker build --build-arg=nginxconf=deploy/kotsadm.conf -f deploy/Dockerfile -t kotsadm/kotsadm-web:alpha .
	docker push kotsadm/kotsadm-web:alpha
