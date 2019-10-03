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
	yarn test:unit

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
		--progress -w --debug --color --env skaffold --mode development --hot \
		--host 0.0.0.0

.PHONY: build-kotsadm
build-kotsadm: SHIP_CLOUD_ENV = enterprise
build-kotsadm: webpack_build

.PHONY: build-replicated-staging
build-replicated-staging: SHIP_CLOUD_ENV = staging
build-replicated-staging: webpack_build

.PHONY: build-replicated-production
build-replicated-production: SHIP_CLOUD_ENV = production
build-replicated-production: webpack_build

.PHONY: build-local
build-local: SHIP_CLOUD_ENV = skaffold
build-local: webpack_build_local

webpack_build_local:
	node --max_old_space_size=6144 ./node_modules/webpack/bin/webpack.js --config webpack.config.js --env ${SHIP_CLOUD_ENV} --mode development

.PHONY: webpack_build
webpack_build:
	SHORT_BUILDKITE_COMMIT = $(shell echo ${BUILDKITE_COMMIT} | cut -c1-7)
	SHIP_CLUSTER_BUILD_VERSION = $(shell echo ${GIT_TAG:-$(SHORT_BUILDKITE_COMMIT)})
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

.PHONY: publish-replicated-staging
publish-replicated-staging:
	docker build --build-arg=nginxconf=deploy/nginx.conf -f deploy/Dockerfile -t replicated/kotsadm-web:alpha .
	docker push replicated/kotsadm-web:alpha

.PHONY: release-replicated-staging
release-replicated-staging: IMAGE = replicated/kotsadm-web:alpha
release-replicated-staging: OVERLAY = staging
release-replicated-staging: GITOPS_OWNER = replicatedcom
release-replicated-staging: GITOPS_REPO = gitops-deploy
release-replicated-staging: GITOPS_BRANCH = master
release-replicated-staging: gitops

.PHONY: publish-replicated-production
publish-replicated-production:
	docker build --build-arg=nginxconf=deploy/nginx.conf -f deploy/Dockerfile -t replicated/kotsadm-web:${GIT_TAG} .
	docker push replicated/kotsadm-web:${GIT_TAG}

.PHONY: release-replicated-production
release-replicated-production: IMAGE = replicated/kotsadm-web:${GIT_TAG}
release-replicated-production: OVERLAY = production
release-replicated-production: GITOPS_OWNER = replicatedcom
release-replicated-production: GITOPS_REPO = gitops-deploy
release-replicated-production: GITOPS_BRANCH = release
release-replicated-production: gitops

gitops:
	cd kustomize/overlays/$(OVERLAY); sed -i -- 's/GIT_SHA_PLACEHOLDER/'$${BUILDKITE_COMMIT:0:7}'/g' *.yaml
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image kotsadm/kotsadm-web=${IMAGE}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/kotsadm-web

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/kotsadm-web/kotsadm-web.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/kotsadm-web; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITOPS_BRANCH)
