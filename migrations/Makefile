SHELL:=/bin/bash
PROJECT_NAME ?= kotsadm-migrations

.PHONY: test
test:
	docker-compose -f docker-compose-citest.yml up -d postgres
	docker-compose -f docker-compose-citest.yml build migrate
	docker-compose -f docker-compose-citest.yml run migrate # forwards exit code
	docker-compose -f docker-compose-citest.yml down

.PHONY: schema-alpha
schema-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
schema-alpha: build_schema

.PHONY: schema-release
schema-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
schema-release: build_schema

build_schema:
	docker pull schemahero/schemahero:alpha
	docker build -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}

.PHONY: publish-release
publish-release: OVERLAY = production
publish-release: GITOPS_OWNER = replicatedcom
publish-release: GITOPS_REPO = gitops-deploy
publish-release: GITOPS_BRANCH = release
publish-release: build_and_publish

.PHONY: publish-alpha
publish-alpha: OVERLAY = staging
publish-alpha: GITOPS_OWNER = replicatedcom
publish-alpha: GITOPS_REPO = gitops-deploy
publish-alpha: GITOPS_BRANCH = master
publish-alpha: build_and_publish

.PHONY: publish-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: build_and_publish

build_and_publish:
	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${PROJECT_NAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITOPS_BRANCH)

