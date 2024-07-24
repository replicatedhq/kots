include Makefile.build.mk
CURRENT_USER := $(if $(GITHUB_USER),$(GITHUB_USER),$(shell id -u -n))
MINIO_TAG ?= 0.20240716.234641-r0
RQLITE_TAG ?= 8.26.7-r0
DEX_TAG ?= 2.40.0-r3
LVP_TAG ?= v0.6.7

define sendMetrics
@if [ -z "${PROJECT_NAME}" ]; then \
    echo "PROJECT_NAME not defined"; \
    exit 1; \
fi
@curl -X POST "https://api.datadoghq.com/api/v1/series" \
-H "Content-Type: text/json" \
-H "DD-API-KEY: ${DD_API_KEY}" \
-d "{\"series\": [{\"metric\": \"build.time\",\"points\": [[$$(date +%s), $$(expr $$(date +%s) - $$(cat start-time))]],\"tags\": [\"service:${PROJECT_NAME}\"]}]}"
endef

.PHONY: capture-start-time
capture-start-time:
	@echo $$(date +%s) > start-time

.PHONY: report-metric
report-metric:
	@$(if ${DD_API_KEY}, $(call sendMetrics))
	@rm start-time

.PHONY: test
test:
	if [ -n "$(RUN)" ]; then \
		go test $(TEST_BUILDFLAGS) ./pkg/... ./cmd/... -coverprofile cover.out -run $(RUN); \
	else \
		go test $(TEST_BUILDFLAGS) ./pkg/... ./cmd/... -coverprofile cover.out; \
	fi

.PHONY: e2e
e2e:
	${MAKE} -C e2e

.PHONY: integration-cli
integration-cli:
	go build ${LDFLAGS} -o bin/kots-integration ./integration

.PHONY: ci-test
ci-test:
	go test $(TEST_BUILDFLAGS) ./pkg/... ./cmd/... ./integration/... -coverprofile cover.out

.PHONY: kots
kots: PROJECT_NAME = kots
kots: capture-start-time kots-real report-metric

.PHONY: kots-real
kots-real:
	mkdir -p web/dist
	touch web/dist/README.md
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

.PHONY: mock
mock:
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -source=pkg/store/store_interface.go -destination=pkg/store/mock/mock.go
	mockgen -source=pkg/handlers/interface.go -destination=pkg/handlers/mock/mock.go
	mockgen -source=pkg/operator/client/client_interface.go -destination=pkg/operator/client/mock/mock.go

.PHONY: build
build: PROJECT_NAME = kotsadm
build: capture-start-time build-real report-metric

.PHONY: build-real
build-real:
	mkdir -p web/dist
	touch web/dist/README.md
	go build ${LDFLAGS} ${GCFLAGS} -v -o bin/kotsadm $(BUILDFLAGS) ./cmd/kotsadm

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run
run:
	./bin/kotsadm api

.PHONY: okteto-dev
okteto-dev:
    ## We download all go modules, instead of putting them in the container. This will
    ## use the PVC that everyone has, and will build a cache.
    ##
    ## We also run `make build` here because the initial compilation is slow and
    ## this enabled `okteto up` to do all of the long-running stuff and give the user
    ## a pretty good env right after
	@go mod download -x
	@make build
	@printf "\n\n To build and run api, run: \n\n   # make build run\n\n"

# Debugging
.PHONY: debug-build
debug-build:
	go build ${LDFLAGS} ${GCFLAGS} $(BUILDFLAGS) -v -o ./bin/kotsadm-debug ./cmd/kotsadm

.PHONY: debug
debug: debug-build
	LOG_LEVEL=$(LOG_LEVEL) dlv --listen=:2345 --headless=true --api-version=2 exec ./bin/kotsadm-debug api

.PHONY: web
web:
	source .image.env && ${MAKE} -C web build-kotsadm

.PHONY: build-ttl.sh
build-ttl.sh: export GOOS ?= linux
build-ttl.sh: export GOARCH ?= amd64
build-ttl.sh: web kots build
	docker build --platform $(GOOS)/$(GOARCH) -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kotsadm:24h .
	docker push ttl.sh/${CURRENT_USER}/kotsadm:24h

.PHONY: all-ttl.sh
all-ttl.sh: export GOOS ?= linux
all-ttl.sh: export GOARCH ?= amd64
all-ttl.sh: build-ttl.sh
	source .image.env && \
		IMAGE=ttl.sh/${CURRENT_USER}/kotsadm-migrations:24h \
		DOCKER_BUILD_ARGS="--platform $(GOOS)/$(GOARCH)" \
		make -C migrations build_schema

	docker pull --platform $(GOOS)/$(GOARCH) kotsadm/minio:${MINIO_TAG}
	docker tag kotsadm/minio:${MINIO_TAG} ttl.sh/${CURRENT_USER}/minio:${MINIO_TAG}
	docker push ttl.sh/${CURRENT_USER}/minio:${MINIO_TAG}

	docker pull --platform $(GOOS)/$(GOARCH) kotsadm/rqlite:${RQLITE_TAG}
	docker tag kotsadm/rqlite:${RQLITE_TAG} ttl.sh/${CURRENT_USER}/rqlite:${RQLITE_TAG}
	docker push ttl.sh/${CURRENT_USER}/rqlite:${RQLITE_TAG}

.PHONY: kotsadm-bundle
kotsadm-bundle:
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/kotsadm:${GIT_TAG} docker://${BUNDLE_REGISTRY}/kotsadm:${GIT_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/kotsadm-migrations:${GIT_TAG} docker://${BUNDLE_REGISTRY}/kotsadm-migrations:${GIT_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/dex:${DEX_TAG} docker://${BUNDLE_REGISTRY}/dex:${DEX_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/minio:${MINIO_TAG} docker://${BUNDLE_REGISTRY}/minio:${MINIO_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/rqlite:${RQLITE_TAG} docker://${BUNDLE_REGISTRY}/rqlite:${RQLITE_TAG}
	skopeo copy --all --dest-tls-verify=false docker://replicated/local-volume-provider:${LVP_TAG} docker://${BUNDLE_REGISTRY}/local-volume-provider:${LVP_TAG}

	go run ./scripts/create-airgap-file.go true

.PHONY: kotsadm-bundle-nominio
kotsadm-bundle-nominio:
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/kotsadm:${GIT_TAG} docker://${BUNDLE_REGISTRY}/kotsadm:${GIT_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/kotsadm-migrations:${GIT_TAG} docker://${BUNDLE_REGISTRY}/kotsadm-migrations:${GIT_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/dex:${DEX_TAG} docker://${BUNDLE_REGISTRY}/dex:${DEX_TAG}
	skopeo copy --all --dest-tls-verify=false docker://kotsadm/rqlite:${RQLITE_TAG} docker://${BUNDLE_REGISTRY}/rqlite:${RQLITE_TAG}
	skopeo copy --all --dest-tls-verify=false docker://replicated/local-volume-provider:${LVP_TAG} docker://${BUNDLE_REGISTRY}/local-volume-provider:${LVP_TAG}

	go run ./scripts/create-airgap-file.go false

.PHONY: cache
cache:
	docker build -f hack/dev/skaffoldcache.Dockerfile . -t kotsadm:cache

.PHONY: init-sbom
init-sbom:
	mkdir -p sbom/spdx

.PHONY: install-spdx-sbom-generator
install-spdx-sbom-generator: init-sbom
ifeq (,$(shell command -v spdx-sbom-generator))
	./scripts/install-sbom-generator.sh
SPDX_GENERATOR=./sbom/spdx-sbom-generator
else
SPDX_GENERATOR=$(shell command -v spdx-sbom-generator)
endif

sbom/spdx/bom-go-mod.spdx: install-spdx-sbom-generator
	$(SPDX_GENERATOR) -o ./sbom/spdx

sbom/kots-sbom.tgz: sbom/spdx/bom-go-mod.spdx
	tar -czf sbom/kots-sbom.tgz sbom/spdx/*.spdx

sbom: sbom/kots-sbom.tgz
	cosign sign-blob -key ./cosign.key sbom/kots-sbom.tgz > ./sbom/kots-sbom.tgz.sig
	cosign public-key -key ./cosign.key -outfile ./sbom/key.pub

# npm packages scans are ignored(only go modules are scanned)
.PHONY: scan
scan:
	trivy fs \
		--scanners vuln \
		--exit-code=1 \
		--severity="CRITICAL,HIGH,MEDIUM" \
		--ignore-unfixed \
		--skip-dirs .github \
		--skip-files actions/version-tag/package-lock.json \
		--skip-files web/yarn.lock \
		--skip-dirs web/node_modules \
		--ignorefile .trivyignore \
		./

.PHONY: generate-kubectl-versions
generate-kubectl-versions:
	node .github/actions/kubectl-versions/dist/index.js
