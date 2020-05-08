SHELL := /bin/bash
PROJECT_NAME ?= kotsadm-fixtures

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
build: prebuild deps
	`yarn bin`/tsc --project .

.PHONY: run
run:
	node --no-deprecation ./build/fixtures.js generate

.PHONY: schema-fixtures
schema-fixtures:
	docker pull schemahero/schemahero:0.7.2
	docker run --user root -e uid=1 -v `pwd`:/out -v `pwd`/../tables:/in schemahero/schemahero:0.7.2 fixtures --input-dir /in --output-dir /out/schema --dbname -ship-cloud --driver postgres

.PHONY: publish
publish: IMAGE = kotsadm/${PROJECT_NAME}:latest
publish:
	docker build -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
