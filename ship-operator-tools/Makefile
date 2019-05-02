.PHONY: docker-build dep-deps deps

FULLSRC = $(shell find . -name "*.go")

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

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
