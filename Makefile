
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

.PHONY: test
test:
	go test ./pkg/... ./cmd/... ./ffi/... -coverprofile cover.out

.PHONY: integration-cli
integration-cli:
	go build -o bin/kots-integration ./integration
	
.PHONY: integration
integration:
	go test -v ./integration/...
	
.PHONY: kots
kots: fmt vet
	go build -o bin/kots github.com/replicatedhq/kots/cmd/kots

.PHONY: ffi
ffi: fmt vet
	go build -o bin/kots.so -buildmode=c-shared ./ffi/...

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
