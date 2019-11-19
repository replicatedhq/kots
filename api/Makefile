SHELL := /bin/bash
PROJECT_NAME ?= kotsadm-api

.PHONY: deps
deps:
	npm i --silent

.PHONY: deps-prod
deps-prod:
	npm i --silent

.PHONY: prebuild
prebuild:
	rm -rf build
	mkdir -p build

.PHONY: lint
lint:
	npx tslint --project ./tsconfig.json --fix

.PHONY: build
build: prebuild
	`npm bin`/tsc --project .

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
test-and-publish:
	npm run test:broker

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
test: deps
	@-docker stop kotsadm-fixtures > /dev/null 2>&1 || :
	@-docker rm -f kotsadm-fixtures > /dev/null 2>&1 || :
	docker run --rm -d --name kotsadm-fixtures -p 15432:5432 kotsadm/kotsadm-fixtures:local
	npm run test:local
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
