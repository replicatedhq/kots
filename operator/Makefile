
SHELL := /bin/bash -o pipefail
PROJECT_NAME ?= kotsadm-operator
export GO111MODULE=on

.PHONY: publish-pact
publish-pact:
	curl \
	--silent --output /dev/null --show-error --fail \
	--user ${PACT_BROKER_USERNAME}:${PACT_BROKER_PASSWORD} \
	-X PUT \
	-H "Content-Type: application/json" \
	-d@pacts/operator-kotsadm-api.json \
	https://replicated-pact-broker.herokuapp.com/pacts/provider/kotsadm-api/consumer/operator/version/0.0.1

.PHONY: pacts
pacts:
	go test ./pkg/pacts

.PHONY: test
test: pacts

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: test
test: fmt vet
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: build
build: fmt vet
	go build -o bin/kotsadm-operator github.com/replicatedhq/kotsadm/operator/cmd/kotsadm-operator

.PHONY: runs
run: bin
	./bin/kotsadm-operator --api-endpoint http://localhost:30065 --token local

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
publish-release: GITOPS_FILENAME = kotsadm-operator
publish-release: build_and_publish

.PHONY: publish-alpha
publish-alpha: IMAGE_TAG = alpha
publish-alpha: OVERLAY = staging
publish-alpha: GITOPS_OWNER = replicatedcom
publish-alpha: GITOPS_REPO = gitops-deploy
publish-alpha: GITOPS_BRANCH = master
publish-alpha: GITOPS_FILENAME = kotsadm-operator
publish-alpha: build_and_publish

# .PHONY: publish-enterprise
# publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
# publish-enterprise: OVERLAY = enterprise
# publish-enterprise: GITOPS_OWNER = replicatedhq
# publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
# publish-enterprise: GITOPS_BRANCH = master
# publish-enterprise: GITOPS_FILENAME = kotsadm-operator
# publish-enterprise: build_and_publish

build_and_publish:
	cd kustomize/overlays/$(OVERLAY); kustomize edit set image kotsadm/${PROJECT_NAME}=kotsadm/${PROJECT_NAME}:${IMAGE_TAG}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${GITOPS_FILENAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
	      git push origin $(GITOPS_BRANCH)
