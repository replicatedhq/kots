SHELL := /bin/bash
PROJECT_NAME ?= kotsadm-web

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
test:
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

.PHONY: build-alpha
build-alpha: SHIP_CLOUD_ENV = staging
build-alpha: webpack_build

.PHONY: build-release
build-release: SHIP_CLOUD_ENV = production
build-release: webpack_build

.PHONY: build-enterprise
build-enterprise: SHIP_CLOUD_ENV = enterprise
build-enterprise: webpack_build

.PHONY: build-local
build-local: SHIP_CLOUD_ENV = skaffold
build-local: webpack_build_local

webpack_build_local:
	node --max_old_space_size=6144 ./node_modules/webpack/bin/webpack.js --config webpack.config.js --env ${SHIP_CLOUD_ENV} --mode development

.PHONY: webpack_build
webpack_build: SHIP_CLUSTER_BUILD_VERSION = $(shell echo ${BUILDKITE_COMMIT} | cut -c1-7)
webpack_build:
	SHIP_CLUSTER_BUILD_VERSION=$(SHIP_CLUSTER_BUILD_VERSION) \
	node \
	--max_old_space_size=6144 \
	./node_modules/webpack/bin/webpack.js \
	--config webpack.config.js \
	--env ${SHIP_CLOUD_ENV} \
	--mode production

.PHONY: publish-release
publish-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
publish-release: OVERLAY = production
publish-release: GITOPS_OWNER = replicatedcom
publish-release: GITOPS_REPO = gitops-deploy
publish-release: GITOPS_BRANCH = release
publish-release: docker gitops

.PHONY: publish-alpha
publish-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
publish-alpha: OVERLAY = staging
publish-alpha: GITOPS_OWNER = replicatedcom
publish-alpha: GITOPS_REPO = gitops-deploy
publish-alpha: GITOPS_BRANCH = master
publish-alpha: docker gitops

.PHONY: publish-enterprise
publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: docker gitops

docker-kotsadm:
	docker build --build-arg=nginxconf=deploy/kotsadm.conf -f deploy/Dockerfile -t kotsadm/kotsadm-web:alpha .
	docker push kotsadm/kotsadm-web:alpha

docker:
	docker build -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}

gitops:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image kotsadm/${PROJECT_NAME}=${IMAGE}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${PROJECT_NAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITOPS_BRANCH)
