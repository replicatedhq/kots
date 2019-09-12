export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

CODEDIRS := ./pkg/... ./cmd/... ./ffi/...
DEFAULT_TEST := fmt vet test
ARTIFACTS := bin/kots bin/kots.so

.PHONY: test
test:
	go test $(CODEDIRS) -coverprofile cover.out

.PHONY: clean
clean:
	for a in $(ARTIFACTS) cover.out; do if test -e $$a; then rm -fv $$a; fi; done

## artifacts

# making the DEFAULT_TEST a prerequisite to ARTIFACTS instead of the
# individual binaries means they only get run once.
.PHONY: all
all: $(DEFAULT_TEST) | $(ARTIFACTS)

.PHONY: kots
kots: $(DEFAULT_TEST) | bin/kots

bin/kots:
	go build -o $@ github.com/replicatedhq/kots/cmd/kots

.PHONY: ffi
ffi: $(DEFAULT_TEST) | bin/kots.so

bin/kots.so:
	go build -o $@ -buildmode=c-shared ./ffi/...

## execute linters

.PHONY: fmt
fmt:
	go fmt $(CODEDIRS)

.PHONY: vet
vet:
	go vet $(CODEDIRS)

.PHONY: gosec
gosec: $(GOPATH)/bin/gosec
	$(GOPATH)/bin/gosec ./...

.PHONY: staticcheck
staticcheck: $(GOPATH)/bin/staticcheck
	$(GOPATH)/bin/staticcheck $(CODEDIRS)

## install linters

# only install staticcheck if it is not present in GOPATH
$(GOPATH)/bin/staticcheck:
	go get honnef.co/go/tools/cmd/staticcheck

# only install gosec if it is not present in GOPATH
$(GOPATH)/bin/gosec:
	go get github.com/securego/gosec/cmd/gosec

## release recipes

.PHONY: snapshot-release
snapshot-release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --snapshot --config deploy/.goreleaser.snapshot.yml

.PHONY: release
release: export GITHUB_TOKEN = $(shell echo ${GITHUB_TOKEN_REPLICATEDBOT})
release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --config deploy/.goreleaser.yml
