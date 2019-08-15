.DEFAULT_GOAL := all

export GO111MODULE=off

SHELL := /bin/bash -o pipefail
PROJECT_NAME ?= kotsadm-worker

SRC = $(shell find . -name "*.go")

VERSION_PACKAGE = github.com/replicatedhq/kotsadm/worker/pkg/version
VERSION=`git describe --tags &>/dev/null || echo "v0.0.1"`
GIT_SHA=`git rev-parse HEAD`
DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

define LDFLAGS
-ldflags "\
	-X ${VERSION_PACKAGE}.version=${VERSION} \
	-X ${VERSION_PACKAGE}.gitSHA=${GIT_SHA} \
	-X ${VERSION_PACKAGE}.buildTime=${DATE} \
"
endef

LOG_LEVEL ?= debug

.PHONY: all
all: vet test build

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: run
run: build
	LOG_LEVEL=$(LOG_LEVEL) ./bin/kotsadm-worker init

.PHONY: build-deps
build-deps:
	hack/get_build_deps.sh
	./hack/get_build_deps.sh

.PHONY: dep-deps
dep-deps:
	go get github.com/golang/dep/cmd/dep

.PHONY: deps
deps:
	dep ensure -v

.PHONY: lint
lint:
	golint ./pkg/... | grep -v "should have comment" | grep -v "comment on exported" | grep -v bindata || :
	golint ./cmd/... | grep -v "should have comment" | grep -v "comment on exported" | grep -v bindata || :

.PHONY: vet
vet: lint
	go vet ./pkg/...
	go vet ./cmd/...

.PHONY: test
test:
	time go test ./pkg/... | grep -v '?'

.PHONY: bin/kotsadm-worker
bin/kotsadm-worker: $(SRC)
	time go build -i \
		${LDFLAGS} \
		-o bin/kotsadm-worker \
		./cmd/kotsadm-worker
	@echo built bin/kotsadm-worker

.PHONY: build
build: bin/kotsadm-worker

.PHONY: build-alpha
build-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
build-alpha: build_and_push

.PHONY: build-release
build-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
build-release: build_and_push

.PHONY: build_and_push
build_and_push:
	docker build -f deploy/Dockerfile -t ${IMAGE} ./deploy
	docker push ${IMAGE}

.PHONY: publish-release
publish-release: IMAGE_TAG = ${GIT_TAG}
publish-release: OVERLAY = production
publish-release: GITOPS_OWNER = replicatedcom
publish-release: GITOPS_REPO = gitops-deploy
publish-release: GITOPS_BRANCH = release
publish-release: GITOPS_FILENAME = kotsadm-worker
publish-release: build_and_publish

.PHONY: publish-alpha
publish-alpha: IMAGE_TAG = alpha
publish-alpha: OVERLAY = staging
publish-alpha: GITOPS_OWNER = replicatedcom
publish-alpha: GITOPS_REPO = gitops-deploy
publish-alpha: GITOPS_BRANCH = master
publish-alpha: GITOPS_FILENAME = kotsadm-worker
publish-alpha: build_and_publish

# .PHONY: publish-enterprise
# publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
# publish-enterprise: OVERLAY = enterprise
# publish-enterprise: GITOPS_OWNER = replicatedhq
# publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
# publish-enterprise: GITOPS_BRANCH = master
# publish-enterprise: GITOPS_FILENAME = kotsadm-worker
# publish-enterprise: build_and_publish

build_and_publish:
	cd kustomize/overlays/$(OVERLAY); sed -i -- 's/GIT_SHA_PLACEHOLDER/'$${BUILDKITE_COMMIT:0:7}'/g' *.yaml
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image kotsadm/${PROJECT_NAME}=kotsadm/${PROJECT_NAME}:${IMAGE_TAG}
	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${GITOPS_FILENAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
	      git push origin $(GITOPS_BRANCH)
