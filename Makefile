include Makefile.build
BUILDFLAGS = -tags "netgo containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp" -installsuffix netgo

.PHONY: test
test:
	go test $(BUILDFLAGS) ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: integration-cli
integration-cli:
	go build ${LDFLAGS} -o bin/kots-integration ./integration

.PHONY: ci-test
ci-test:
	go test $(BUILDFLAGS) ./pkg/... ./cmd/... ./integration/... -coverprofile cover.out

.PHONY: kots
kots: fmt vet
	go build ${LDFLAGS} -o bin/kots $(BUILDFLAGS) github.com/replicatedhq/kots/cmd/kots

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet $(BUILDFLAGS) ./pkg/... ./cmd/...

.PHONY: gosec
gosec:
	go get github.com/securego/gosec/cmd/gosec
	$(GOPATH)/bin/gosec ./...

.PHONY: release
release:
	curl -sL https://git.io/goreleaser | VERSION=v0.118.2 bash -s -- --rm-dist --config deploy/.goreleaser.yml
