
SHELL := /bin/bash -o pipefail
IMG ?= replicated/ship-cd:latest
KUBEBUILDER_VERSION ?= 1.0.8
PROJECT_NAME ?= ship-cd

ci: test manager

publish-pact:
	mkdir -p pacts; buildkite-agent artifact download pacts/* pacts --step test-and-build
	curl \
	--silent --output /dev/null --show-error --fail \
	--user ${PACT_BROKER_USERNAME}:${PACT_BROKER_PASSWORD} \
	-X PUT \
	-H "Content-Type: application/json" \
	-d@pacts/ship-cd-ship-cluster-api.json \
	https://replicated-pact-broker.herokuapp.com/pacts/provider/ship-cluster-api/consumer/ship-cd/version/0.0.1

publish: docker-build docker-push

all: test manager

pacts:
	go test ./pkg/controller/cluster/pacts

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/replicatedhq/ship-cluster/ship-cd/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

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
	rm -r ./pkg/client/shipclusterclientset/fake

# Build the docker image
docker-build:
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}

.PHONY: build-enterprise
build-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
build-enterprise: build_and_push

build_and_push:
	# Fetch the artifacts from the previous step
	mkdir -p bin; buildkite-agent artifact download bin/* bin --step test-and-build; chmod +x bin/manager

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

.PHONY: test
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: test-pact
test-pact: generate fmt vet manifests
	go test ./pkg/controller/cluster/pacts/...
