
export GO111MODULE=on

.PHONY: test
test:
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: kots
kots: fmt vet
	go build -o bin/kots github.com/replicatedhq/kots/cmd/kots

.PHONY: ffi
ffi: fmt vet
	go build -o bin/kots.so -buildmode=c-shared ffi/main.go

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

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
