include ./Makefile.build

CURRENT_USER := $(shell id -u -n)

.PHONY: test
test:
	go test ./cmd/...

.PHONY: build
build:
	go build ${LDFLAGS} -o bin/kurl_proxy cmd/main.go

.PHONY: run
run:
	./bin/kurl_proxy

.PHONY: build-ttl.sh
build-ttl.sh: export GOOS ?= linux
build-ttl.sh: export GOARCH ?= amd64
build-ttl.sh: build
	docker build --platform $(GOOS)/$(GOARCH) --pull -f ../dev/dockerfiles/kurl-proxy/Dockerfile.ttlsh -t ttl.sh/${CURRENT_USER}/kurl-proxy:24h .
	docker push ttl.sh/${CURRENT_USER}/kurl-proxy:24h
