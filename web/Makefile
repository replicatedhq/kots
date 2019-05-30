SHELL := /bin/bash
PROJECT_NAME ?= ship-cluster-ui

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
	rm -rf pacts/ship-cluster-ui-ship-cluster-api.json
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
		--progress -w --debug --color --env skaffold --mode development --hot \
		--host 0.0.0.0

.PHONY: build-cache
build-cache:
	docker build -f Dockerfile.skaffoldcache -t repldev/${PROJECT_NAME}:latest .

.PHONY: publish-cache
publish-cache:
	docker push repldev/${PROJECT_NAME}:latest

.PHONY: build-staging
build-staging: SHIP_CLOUD_ENV = staging
build-staging: webpack_build

.PHONY: build-production
build-production: SHIP_CLOUD_ENV = production
build-production: webpack_build

.PHONY: build-enterprise
build-enterprise: SHIP_CLOUD_ENV = enterprise
build-enterprise: webpack_build

webpack_build:
	node --max_old_space_size=6144 ./node_modules/webpack/bin/webpack.js --config webpack.config.js --env ${SHIP_CLOUD_ENV} --mode production

.PHONY: publish-production
publish-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
publish-production: OVERLAY = production
publish-production: GITOPS_OWNER = replicatedcom
publish-production: GITOPS_REPO = gitops-deploy
publish-production: GITOPS_BRANCH = release
publish-production: docker
publish-production: gitops

.PHONY: publish-staging
publish-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
publish-staging: OVERLAY = staging
publish-staging: GITOPS_OWNER = replicatedcom
publish-staging: GITOPS_REPO = gitops-deploy
publish-staging: GITOPS_BRANCH = master
publish-staging: docker
publish-staging: gitops

.PHONY: publish-enterprise
publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: docker
publish-enterprise: gitops

docker:
	docker build -f deploy/Dockerfile -t ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} .
	docker tag ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

gitops:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image $(REGISTRY)/${PROJECT_NAME}=$(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${PROJECT_NAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITOPS_BRANCH)



