include ./Makefile.build
CURRENT_USER := $(shell id -u -n)

.PHONY: test
test:
	go test ./cmd/...

.PHONY: build
build:
	go build ${LDFLAGS} -o bin/kurl_proxy cmd/main.go

.PHONY: up
up:
	skaffold dev -f skaffold.yaml

.PHONY: build-alpha
build-alpha:
	docker build --pull -f deploy/Dockerfile -t kotsadm/kurl-proxy:alpha .
	docker push kotsadm/kurl-proxy:alpha

.PHONY: build-release
build-release:
	docker build --pull -f deploy/Dockerfile -t kotsadm/kurl-proxy:${GIT_TAG} .
	docker push kotsadm/kurl-proxy:${GIT_TAG}
	docker tag kotsadm/kurl-proxy:${GIT_TAG} kotsadm/kurl-proxy:v0.0.0-nightly
	docker push kotsadm/kurl-proxy:v0.0.0-nightly

.PHONY: build-ttl.sh
build-ttl.sh: build
	docker build --pull -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kurl-proxy:12h .
	docker push ttl.sh/${CURRENT_USER}/kurl-proxy:12h
