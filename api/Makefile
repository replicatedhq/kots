SHELL := /bin/bash
PROJECT_NAME ?= ship-cluster-api

.PHONY: deps
deps:
	yarn --silent --frozen-lockfile

.PHONY: prebuild
prebuild:
	rm -rf build
	mkdir -p build

.PHONY: lint
lint:
	npx tslint --project ./tsconfig.json --fix

.PHONY: build
build: prebuild
	`yarn bin`/tsc --project .

.PHONY: run
run:
	node --no-deprecation ./build/server/index.js

.PHONY: run-debug
run-debug:
	node --inspect=0.0.0.0 -r ts-node/register --no-deprecation ./src/server/index.ts

.PHONY: can-i-deploy
can-i-deploy:
	pact-broker can-i-deploy --pacticipant ship-cluster-api --broker-base-url https://replicated-pact-broker.herokuapp.com --latest

.PHONY: test-and-publish
test-and-publish: export POSTGRES_URI = postgresql://shipcloud:password@postgres:5432/shipcloud?connect_timeout=10&application_name=ship&sslmode=disable
test-and-publish: export SESSION_KEY = testsession
test-and-publish: export SKIP_DEPLOY_TO_WORKER = true
test-and-publish: export OBJECT_STORE_IN_DATABASE = true
test-and-publish: export PUBLISH_PACT_VERIFICATION = true
test-and-publish: export S3_ENDPOINT = http://localhost:14569/
test-and-publish: export S3_ACCESS_KEY_ID = S3RVER
test-and-publish: export S3_SECRET_ACCESS_KEY = S3RVER
test-and-publish: export S3_BUCKET_ENDPOINT = true
test-and-publish: export S3_BUCKET_NAME = ship-pacts
test-and-publish:
	yarn run test:broker

.PHONY: test
test: export POSTGRES_URI = postgresql://shipcloud:password@localhost:15432/shipcloud?connect_timeout=10&application_name=ship&sslmode=disable
test: export SESSION_KEY = testsession
test: export SKIP_DEPLOY_TO_WORKER = true
test: export OBJECT_STORE_IN_DATABASE = true
test: export S3_ENDPOINT = http://localhost:14569/
test: export S3_ACCESS_KEY_ID = S3RVER
test: export S3_SECRET_ACCESS_KEY = S3RVER
test: export S3_BUCKET_ENDPOINT = true
test: export S3_BUCKET_NAME = ship-pacts
test:
	@-docker stop ship-cluster-fixtures > /dev/null 2>&1 || :
	@-docker rm -f ship-cluster-fixtures > /dev/null 2>&1 || :
	docker run --rm -d --name ship-cluster-fixtures -p 15432:5432 replicated/ship-cluster-fixtures:local
	yarn run test:local
	@-sleep 1
	docker stop ship-cluster-fixtures
	@-docker rm -f ship-cluster-fixtures > /dev/null 2>&1 || :

.PHONY: pkg
pkg:
	mkdir -p bin
	npx pkg -t node8.11.3 --options no-deprecation --output bin/ship-cluster-api .
	npx pkg -t node8.11.3 --options no-deprecation --output bin/shipcloudctl -c ./package.json ./build/api.js

.PHONY: build-cache
build-cache:
	docker build -f Dockerfile.skaffoldcache -t repldev/${PROJECT_NAME}:latest .

.PHONY: publish-cache
publish-cache:
	docker push repldev/${PROJECT_NAME}:latest

.PHONY: build-staging
build-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
build-staging: build_and_push

.PHONY: build-production
build-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
build-production: build_and_push

.PHONY: build-enterprise
build-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
build-enterprise: build_and_push

build_and_push:
	docker build -f deploy/Dockerfile -t ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} .
	docker tag ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}


.PHONY: publish-production
publish-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
publish-production: OVERLAY = production
publish-production: GITOPS_OWNER = replicatedcom
publish-production: GITOPS_REPO = gitops-deploy
publish-production: GITOPS_BRANCH = release
publish-production: GITOPS_FILENAME = ship-cluster-api
publish-production: build_and_publish

.PHONY: publish-staging
publish-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
publish-staging: OVERLAY = staging
publish-staging: GITOPS_OWNER = replicatedcom
publish-staging: GITOPS_REPO = gitops-deploy
publish-staging: GITOPS_BRANCH = master
publish-staging: GITOPS_FILENAME = ship-cluster-api
publish-staging: build_and_publish

.PHONY: publish-enterprise
publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: GITOPS_FILENAME = ship-cluster-api
publish-enterprise: build_and_publish

.PHONY: publish-enterprise-aka
publish-enterprise-aka: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise-aka: OVERLAY = enterprise-aka
publish-enterprise-aka: GITOPS_OWNER = replicatedhq
publish-enterprise-aka: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise-aka: GITOPS_BRANCH = master
publish-enterprise-aka: GITOPS_FILENAME = ship-cluster-api-aka
publish-enterprise-aka: build_and_publish

build_and_publish:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image $(REGISTRY)/${PROJECT_NAME}=$(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${GITOPS_FILENAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITOPS_BRANCH)

