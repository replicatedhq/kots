
export GO111MODULE=on

.PHONY: test
test:
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: kots
kots: fmt vet
	go build -o bin/kots github.com/replicatedhq/kots/cmd/kots

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: snapshot-release
snapshot-release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --snapshot --config deploy/.goreleaser.snapshot.yml

