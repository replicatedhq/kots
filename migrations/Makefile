SHELL:=/bin/bash
PROJECT_NAME ?= ship-cluster-migrations

.PHONY: test
test:
	docker-compose -f docker-compose-citest.yml up -d postgres
	docker-compose -f docker-compose-citest.yml build migrate
	docker-compose -f docker-compose-citest.yml run migrate # forwards exit code
	docker-compose -f docker-compose-citest.yml down

.PHONY: new-migration
new-migration:
	docker run -v ${PWD}/migrations/:/migrations -it --rm migrate/migrate create -seq -dir /migrations -ext=sql $(name)
	rm migrations/*_$(name).down.sql

.PHONY: check-dups
check-dups:
	@duplicates=`ls migrations/ | grep '[0-9]*_.*\.up' | grep -o '[0-9][0-9]*' | uniq -d`; \
	\
	if [ -z "$$duplicates" ]; then \
	  echo "No duplicates found"; \
	else \
	  echo "Duplicates were found!"; \
	  echo "duplicates - $$duplicates"; \
	  exit 1; \
	fi

.PHONY: build-staffold-cache
build-skaffold-cache:
	docker build -f Dockerfile.skaffold -t repldev/${PROJECT_NAME}:latest .
	docker push repldev/${PROJECT_NAME}:latest
	
.PHONY: publish-production
publish-production: REGISTRY = 799720048698.dkr.ecr.us-east-1.amazonaws.com
publish-production: OVERLAY = production
publish-production: GITOPS_OWNER = replicatedcom
publish-production: GITOPS_REPO = gitops-deploy
publish-production: GITOPS_BRANCH = release
publish-production: build_and_publish

.PHONY: publish-staging
publish-staging: REGISTRY = 923411875752.dkr.ecr.us-east-1.amazonaws.com
publish-staging: OVERLAY = staging
publish-staging: GITOPS_OWNER = replicatedcom
publish-staging: GITOPS_REPO = gitops-deploy
publish-staging: GITOPS_BRANCH = master
publish-staging: build_and_publish

.PHONY: publish-enterprise
publish-enterprise: REGISTRY = registry.replicated.com/ship-enterprise
publish-enterprise: OVERLAY = enterprise
publish-enterprise: GITOPS_OWNER = replicatedhq
publish-enterprise: GITOPS_REPO = ship-enterprise-deploy
publish-enterprise: GITOPS_BRANCH = master
publish-enterprise: build_and_publish

build_and_publish:
	docker build -f deploy/Dockerfile -t ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} .
	docker tag ${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7} $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}
	docker push $(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

	cd kustomize/overlays/$(OVERLAY); kustomize edit set image $(REGISTRY)/${PROJECT_NAME}=$(REGISTRY)/${PROJECT_NAME}:$${BUILDKITE_COMMIT:0:7}

	rm -rf deploy/$(OVERLAY)/work
	mkdir -p deploy/$(OVERLAY)/work; cd deploy/$(OVERLAY)/work; git clone --single-branch -b $(GITOPS_BRANCH) git@github.com:$(GITOPS_OWNER)/$(GITOPS_REPO)
	mkdir -p deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}

	kustomize build kustomize/overlays/$(OVERLAY) > deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}/${PROJECT_NAME}.yaml

	cd deploy/$(OVERLAY)/work/$(GITOPS_REPO)/${PROJECT_NAME}; \
	  git add . ;\
	  git commit --allow-empty -m "$${BUILDKITE_BUILD_URL}"; \
          git push origin $(GITHUB_BRANCH)
