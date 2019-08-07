.DEFAULT_GOAL := all

export GO111MODULE=off

SHELL := /bin/bash -o pipefail
PROJECT_NAME ?= ship-cluster-worker

SRC = $(shell find . -name "*.go")

VERSION_PACKAGE = github.com/replicatedhq/ship-cluster/worker/pkg/version
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
	LOG_LEVEL=$(LOG_LEVEL) ./bin/ship-cluster-worker init

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

.PHONY: bin/ship-cluster-worker
bin/ship-cluster-worker: $(SRC)
	time go build -i \
		${LDFLAGS} \
		-o bin/ship-cluster-worker \
		./cmd/ship-cluster-worker
	@echo built bin/ship-cluster-worker

.PHONY: build
build: bin/ship-cluster-worker

.PHONY: build-staging
build-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
build-staging: build_and_push

.PHONY: build-production
build-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
build-production: build_and_push

.PHONY: build-enterprise
build-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
build-enterprise: build_and_push

build_and_push:
	docker build -t ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} ./deploy
	docker tag ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

.PHONY: publish-production
publish-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
publish-production: OVERLAY = production
publish-production: GITOPS_OWNER = replicatedcom
publish-production: GITOPS_REPO = gitops-deploy
publish-production: GITOPS_BRANCH = release
publish-production: GITOPS_FILENAME = ship-cluster-worker
publish-production: build_and_publish

.PHONY: publish-staging
publish-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
publish-staging: OVERLAY = staging
publish-staging: GITOPS_OWNER = replicatedcom
publish-staging: GITOPS_REPO = gitops-deploy
publish-staging: GITOPS_BRANCH = master
publish-staging: GITOPS_FILENAME = ship-cluster-worker
publish-staging: build_and_publish

.PHONY: publish-enterprise
publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: GITOPS_FILENAME = ship-cluster-worker
publish-enterprise: build_and_publish

.PHONY: publish-enterprise-aka
publish-enterprise-aka: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise-aka: OVERLAY = enterprise-aka
publish-enterprise-aka: GITOPS_OWNER = replicatedhq
publish-enterprise-aka: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise-aka: GITOPS_BRANCH = master
publish-enterprise-aka: GITOPS_FILENAME = ship-cluster-worker-aka
publish-enterprise-aka: build_and_publish

build_and_publish:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image $(REGISTRY)/${PROJECT_NAME}=$(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${GITOPS_FILENAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
	      git push origin $(GITOPS_BRANCH)

.PHONY: build-cache
build-cache:
	docker build -f Dockerfile.skaffoldcache -t repldev/${PROJECT_NAME}:latest .
