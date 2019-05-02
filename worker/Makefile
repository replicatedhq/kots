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

.PHONY: clean
clean:
	rm -rf .state/ bin/

.PHONY: run
run: build
	LOG_LEVEL=$(LOG_LEVEL) ./bin/ship-cluster-worker init

.PHONY: .state/build-deps
.state/build-deps: hack/get_build_deps.sh
	./hack/get_build_deps.sh
	@mkdir -p .state/
	@touch .state/build-deps

.PHONY: build-deps
build-deps: .state/build-deps

.PHONY: dep-deps
dep-deps:
	go get github.com/golang/dep/cmd/dep

.PHONY: deps
deps:
	dep ensure -v

.PHONY: .state/fmt
.state/fmt: $(SRC)
	goreturns -w pkg
	goreturns -w cmd
	@mkdir -p .state
	@touch .state/fmt

.PHONY: fmt
fmt: build-deps .state/fmt

.PHONY: .state/lint
.state/lint: $(SRC)
	golint ./pkg/... | grep -v "should have comment" | grep -v "comment on exported" | grep -v bindata || :
	golint ./cmd/... | grep -v "should have comment" | grep -v "comment on exported" | grep -v bindata || :
	@mkdir -p .state
	@touch .state/lint

.PHONY: lint
lint: fmt .state/lint

.PHONY: .state/vet
.state/vet: $(SRC)
	go vet ./pkg/...
	go vet ./cmd/...
	@mkdir -p .state
	@touch .state/vet

.PHONY: vet
vet: lint .state/vet

.PHONY: .state/test
.state/test: $(SRC)
	go test ./pkg/... | grep -v '?'
	@mkdir -p .state
	@touch .state/test

.PHONY: test
test: .state/test

.PHONY: bin/ship-cluster-worker
bin/ship-cluster-worker: $(SRC)
	go build \
		${LDFLAGS} \
		-i \
		-o bin/ship-cluster-worker \
		./cmd/ship-cluster-worker
	@echo built bin/ship-cluster-worker

.PHONY: build
build: test bin/ship-cluster-worker

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
	docker build -f deploy/Dockerfile -t ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} .
	docker tag ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}


# .PHONY: publish-production
# publish-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
# publish-production: OVERLAY = production
# publish-production: GITOPS_OWNER = replicatedcom
# publish-production: GITOPS_REPO = gitops-deploy
# publish-production: GITOPS_BRANCH = release
# publish-production: build_and_publish

# .PHONY: publish-staging
# publish-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
# publish-staging: OVERLAY = staging
# publish-staging: GITOPS_OWNER = replicatedcom
# publish-staging: GITOPS_REPO = gitops-deploy
# publish-staging: GITOPS_BRANCH = master
# publish-staging: build_and_publish

.PHONY: publish-enterprise
publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: build_and_publish

build_and_publish:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image $(REGISTRY)/${PROJECT_NAME}=$(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${PROJECT_NAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITHUB_BRANCH)

