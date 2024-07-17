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

.PHONY: build-ttl.sh
build-ttl.sh: export GOOS ?= linux
build-ttl.sh: export GOARCH ?= amd64
build-ttl.sh: build
	docker build --platform $(GOOS)/$(GOARCH) --pull -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kurl-proxy:24h .
	docker push ttl.sh/${CURRENT_USER}/kurl-proxy:24h
