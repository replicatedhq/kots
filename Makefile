
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

SHELL := /bin/bash -o pipefail
VERSION_PACKAGE = github.com/replicatedhq/kots/pkg/version
VERSION ?=`git describe --tags`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

GIT_TREE = $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifneq "$(GIT_TREE)" ""
define GIT_UPDATE_INDEX_CMD
git update-index --assume-unchanged
endef
define GIT_SHA
`git rev-parse HEAD`
endef
else
define GIT_UPDATE_INDEX_CMD
echo "Not a git repo, skipping git update-index"
endef
define GIT_SHA
""
endef
endif

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

.PHONY: test
test:
	go test ./pkg/... ./cmd/... ./ffi/... -coverprofile cover.out

.PHONY: integration-cli
integration-cli:
	go build -o bin/kots-integration ./integration
	
.PHONY: ci-test
ci-test:
	go test ./pkg/... ./cmd/... ./ffi/... ./integration/... -coverprofile cover.out

	
.PHONY: kots
kots: fmt vet
	go build ${LDFLAGS} -o bin/kots github.com/replicatedhq/kots/cmd/kots

.PHONY: ffi
ffi: fmt vet
	go build ${LDFLAGS} -o bin/kots.so -buildmode=c-shared ./ffi/...

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/... ./ffi/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/... ./ffi/...

.PHONY: gosec
gosec:
	go get github.com/securego/gosec/cmd/gosec
	$(GOPATH)/bin/gosec ./...

.PHONY: snapshot-release
snapshot-release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --snapshot --config deploy/.goreleaser.snapshot.yml

.PHONY: release
release: export GITHUB_TOKEN = $(shell echo ${GITHUB_TOKEN_REPLICATEDBOT})
release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --config deploy/.goreleaser.yml
