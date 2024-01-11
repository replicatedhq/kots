include ./Makefile.build

CURRENT_USER := $(if $(GITHUB_USER),$(GITHUB_USER),$(shell id -u -n))

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

.PHONY: build-ttl.sh
build-ttl.sh: build
	docker build --pull -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kurl-proxy:24h .
	docker push ttl.sh/${CURRENT_USER}/kurl-proxy:24h
