SHELL := /bin/bash -o pipefail
CURRENT_USER := $(shell id -u -n)
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

.PHONY: kotsadm
kotsadm:
	go build -o bin/kotsadm github.com/replicatedhq/kotsadm/cmd/kotsadm

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: test
test: fmt vet
	go test ./pkg/... ./cmd/...

.PHONY: build-ttl.sh
build-ttl.sh:
	docker build -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kotsadm:12h .
	docker push ttl.sh/${CURRENT_USER}/kotsadm:12h

.PHONY: build-alpha
build-alpha:
	docker build -f deploy/Dockerfile -t kotsadm/kotsadm:alpha .
	docker push kotsadm/kotsadm:alpha

.PHONY: build-release
build-release:
	docker build -f deploy/Dockerfile -t kotsadm/kotsadm:${BUILDKITE_TAG} .
	docker push kotsadm/kotsadm:${BUILDKITE_TAG}

.PHONY: project-pact-tests
project-pact-tests:
	make -C web test
	make -C operator test

	make -C migrations/fixtures schema-fixtures build run
	cd migrations && docker build -t kotsadm/kotsadm-fixtures:local -f ./fixtures/deploy/Dockerfile ./fixtures

	mkdir -p api/pacts
	cp web/pacts/kotsadm-web-kotsadm-api.json api/pacts/
	make -C api test

	@echo All contract tests have passed.
