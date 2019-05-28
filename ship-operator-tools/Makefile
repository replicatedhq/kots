.PHONY: docker-build dep-deps deps
PROJECT_NAME ?= ship-operator-tools
FULLSRC = $(shell find . -name "*.go")

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

.PHONY: build-cache
build-cache:
	docker build -f Dockerfile.skaffoldcache -t repldev/${PROJECT_NAME}:latest .

.PHONY: publish-cache
publish-cache:
	docker push repldev/${PROJECT_NAME}:latest

docker-build:
	docker build -t replicated/ship-operator-tools -f Dockerfile.skaffold .

test:
	go test ./pkg/...

dep-deps:
	go get -u github.com/golang/dep/cmd/dep

deps:
	dep ensure -v

build: fmt bin/ship-operator-tools

bin/ship-operator-tools: $(FULLSRC)
	go build \
		${LDFLAGS} \
		-i \
		-o bin/ship-operator-tools \
		./cmd/ship-operator-tools
	@echo built bin/ship-operator-tools

githooks:
	echo 'make fmt' > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit

.state/fmt: $(SRC)
	goimports -w pkg
	goimports -w cmd
	@mkdir -p .state
	@touch .state/fmt

fmt: .state/build-deps .state/fmt

.state/build-deps: hack/get_build_deps.sh
	./hack/get_build_deps.sh
	@mkdir -p .state/
	@touch .state/build-deps

build-deps: .state/build-deps

webhookserver:
	go run cmd/webhook-server/main.go
