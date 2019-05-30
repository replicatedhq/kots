SHELL := /bin/bash -o pipefail
IMG ?= replicated/ship-operator:latest
PROJECT_NAME ?= ship-operator

all: test bin/ship-operator

ci: bin/ship-operator

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build ship-operator binary
bin/ship-operator: generate fmt vet
	go build -o bin/ship-operator ./cmd/ship-operator

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/ship-operator/main.go manager

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...
	rm -r ./pkg/client/shipwatchclientset/fake

# Build the docker image
docker-build: test
	docker build . -t ${IMG} -f Dockerfile.skaffold
	# @echo "updating kustomize image patch file for manager resource"
	# sed -i 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}

githooks:
	echo 'make install' > .git/hooks/post-merge
	chmod +x .git/hooks/post-merge

.PHONY: build-cache
build-cache:
	docker build -f Dockerfile.skaffoldcache -t repldev/${PROJECT_NAME}:latest .

.PHONY: publish-cache
publish-cache:
	docker push repldev/${PROJECT_NAME}:latest

.PHONY: build-enterprise
build-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
build-enterprise: build_and_push

build_and_push:
	# Fetch the artifacts from the previous step
	mkdir -p bin; buildkite-agent artifact download bin/* bin --step test-and-build; chmod +x bin/ship-operator

	docker build -f deploy/Dockerfile -t ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} .
	docker tag ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

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
          git push origin $(GITOPS_BRANCH)

