SHELL := /bin/bash
PROJECT_NAME ?= kotsadm-api

.PHONY: deps
deps:
	yarn --silent --frozen-lockfile

.PHONY: deps-prod
deps-prod:
	yarn --silent --frozen-lockfile --production=true

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
	pact-broker can-i-deploy --pacticipant kotsadm-api --broker-base-url https://replicated-pact-broker.herokuapp.com --latest

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
test-and-publish: export ENABLE_SHIP = 1
test-and-publish: export ENABLE_KOTS = 1
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
test: export ENABLE_SHIP = 1
test: export ENABLE_KOTS = 1
test: deps
	@-docker stop kotsadm-fixtures > /dev/null 2>&1 || :
	@-docker rm -f kotsadm-fixtures > /dev/null 2>&1 || :
	docker run --rm -d --name kotsadm-fixtures -p 15432:5432 kotsadm/kotsadm-fixtures:local
	yarn run test:local
	@-sleep 1
	docker stop kotsadm-fixtures
	@-docker rm -f kotsadm-fixtures > /dev/null 2>&1 || :

.PHONY: build-alpha
build-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
build-alpha: build_and_push

.PHONY: build-release
build-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
build-release: build_and_push

.PHONY: build_and_push
build_and_push:
	docker build -f deploy/Dockerfile -t ${IMAGE} --build-arg commit="$${BUILDKITE_COMMIT:0:7}" .
	docker push ${IMAGE}

.PHONY: publish-release
publish-release: IMAGE_TAG = ${GIT_TAG}
publish-release: OVERLAY = production
publish-release: GITOPS_OWNER = replicatedcom
publish-release: GITOPS_REPO = gitops-deploy
publish-release: GITOPS_BRANCH = release
publish-release: GITOPS_FILENAME = kotsadm-api
#publish-release: build_and_publish

.PHONY: publish-alpha
publish-alpha: IMAGE_TAG = alpha
publish-alpha: OVERLAY = staging
publish-alpha: GITOPS_OWNER = replicatedcom
publish-alpha: GITOPS_REPO = gitops-deploy
publish-alpha: GITOPS_BRANCH = master
publish-alpha: GITOPS_FILENAME = kotsadm-api
publish-alpha: build_and_publish

build_and_publish:
	cd kustomize/overlays/$(OVERLAY); sed -i -- 's/GIT_SHA_PLACEHOLDER/'$${BUILDKITE_COMMIT:0:7}'/g' *.yaml
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image kotsadm/${PROJECT_NAME}=kotsadm/${PROJECT_NAME}:${IMAGE_TAG}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${GITOPS_FILENAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITOPS_BRANCH)
