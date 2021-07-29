include Makefile.build
CURRENT_USER := $(shell id -u -n)
MINIO_VERSION := RELEASE.2021-07-27T02-40-15Z
POSTGRES_VERSION := 10.17-alpine

BUILDFLAGS = -tags='netgo containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp' -installsuffix netgo

.PHONY: test
test:
	go test $(BUILDFLAGS) ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: integration-cli
integration-cli:
	go build ${LDFLAGS} -o bin/kots-integration ./integration

.PHONY: ci-test
ci-test:
	go test $(BUILDFLAGS) ./pkg/... ./cmd/... ./integration/... -coverprofile cover.out

.PHONY: kots
kots: fmt vet
	go build ${LDFLAGS} -o bin/kots $(BUILDFLAGS) github.com/replicatedhq/kots/cmd/kots

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet $(BUILDFLAGS) ./pkg/... ./cmd/...

.PHONY: gosec
gosec:
	go get github.com/securego/gosec/cmd/gosec
	$(GOPATH)/bin/gosec ./...

.PHONY: release
release:
	curl -sL https://git.io/goreleaser | VERSION=v0.118.2 bash -s -- --rm-dist --config deploy/.goreleaser.yml

.PHONY: mock
mock:
	go get github.com/golang/mock/mockgen@v1.4.4
	mockgen -source=pkg/store/store_interface.go -destination=pkg/store/mock/mock.go
	mockgen -source=pkg/handlers/interface.go -destination=pkg/handlers/mock/mock.go

.PHONY: kotsadm
kotsadm:
	go build ${LDFLAGS} -o bin/kotsadm $(BUILDFLAGS) ./cmd/kotsadm

.PHONY: build-ttl.sh
build-ttl.sh: 
	docker build --pull -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kotsadm:12h .
	docker push ttl.sh/${CURRENT_USER}/kotsadm:12h

.PHONY: all-ttl.sh
all-ttl.sh: kotsadm
	make -C web build-kotsadm
	make build-ttl.sh

	IMAGE=ttl.sh/${CURRENT_USER}/kotsadm-migrations:12h make -C migrations build_schema

	make -C kotsadm/operator build
	make -C kotsadm/operator build-ttl.sh

	docker pull minio/minio:${MINIO_VERSION} 
	docker tag minio/minio:${MINIO_VERSION} ttl.sh/${CURRENT_USER}/minio:12h 
	docker push ttl.sh/${CURRENT_USER}/minio:12h  

	docker pull postgres:${POSTGRES_VERSION} 
	docker tag postgres:${POSTGRES_VERSION} ttl.sh/${CURRENT_USER}/postgres:12h
	docker push ttl.sh/${CURRENT_USER}/postgres:12h

.PHONY: build-alpha
build-alpha:
	docker build --pull -f deploy/Dockerfile --build-arg version=${GIT_COMMIT} -t kotsadm/kotsadm:alpha .
	docker push kotsadm/kotsadm:alpha

.PHONY: build-release
build-release:
	docker build --pull -f deploy/Dockerfile --build-arg version=${GIT_TAG} -t kotsadm/kotsadm:${GIT_TAG} .
	docker push kotsadm/kotsadm:${GIT_TAG}
	mkdir -p bin/docker-archive/kotsadm
	skopeo copy docker-daemon:kotsadm/kotsadm:${GIT_TAG} docker-archive:bin/docker-archive/kotsadm/${GIT_TAG}

	docker pull ghcr.io/dexidp/dex:v2.28.1
	docker tag ghcr.io/dexidp/dex:v2.28.1 kotsadm/dex:v2.28.1
	docker push kotsadm/dex:v2.28.1

	mkdir -p bin/docker-archive/dex
	skopeo copy docker://kotsadm/dex:v2.28.1 docker-archive:bin/docker-archive/dex/v2.28.1

	mkdir -p bin/docker-archive/minio
	skopeo copy docker://minio/minio:${MINIO_VERSION} docker-archive:bin/docker-archive/minio/${MINIO_VERSION}

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

.PHONY: cache
cache:
	docker build -f hack/dev/Dockerfile.skaffoldcache . -t kotsadm:cache
	docker build -f kotsadm/operator/Dockerfile.skaffoldcache kotsadm/operator -t kotsadm-operator:cache
