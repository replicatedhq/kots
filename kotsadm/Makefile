include ../Makefile.build
CURRENT_USER := $(shell id -u -n)
BUILDTAGS = netgo containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp

.PHONY: kotsadm
kotsadm:
	go build ${LDFLAGS} -tags "$(BUILDTAGS)" -o bin/kotsadm ./cmd/kotsadm

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet -tags "$(BUILDTAGS)" ./pkg/... ./cmd/...

.PHONY: test
test: fmt vet
	go test -tags "$(BUILDTAGS)" ./pkg/... ./cmd/...

.PHONY: build-ttl.sh
build-ttl.sh:
	docker build -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kotsadm:12h .
	docker push ttl.sh/${CURRENT_USER}/kotsadm:12h

.PHONY: build-alpha
build-alpha:
	docker build -f deploy/Dockerfile --build-arg version=${GIT_COMMIT} -t kotsadm/kotsadm:alpha .
	docker push kotsadm/kotsadm:alpha

.PHONY: build-release
build-release:
	docker build -f deploy/Dockerfile --build-arg version=${GIT_TAG} -t kotsadm/kotsadm:${GIT_TAG} .
	docker push kotsadm/kotsadm:${GIT_TAG}
	mkdir -p bin/docker-archive/kotsadm
	skopeo copy docker-daemon:kotsadm/kotsadm:${GIT_TAG} docker-archive:bin/docker-archive/kotsadm/${GIT_TAG}
	mkdir -p bin/docker-archive/dex
	skopeo copy docker://quay.io/dexidp/dex:v2.26.0 docker-archive:bin/docker-archive/dex/${GIT_TAG}

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
