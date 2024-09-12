include Makefile.build.mk
CURRENT_USER := $(shell id -u -n)
MINIO_TAG ?= 0.20240909.165928-r0
RQLITE_TAG ?= 8.30.2-r0
DEX_TAG ?= 2.41.1-r0
LVP_TAG ?= v0.6.7
PACT_PUBLISH_CONTRACT ?= false

OS ?= linux
ARCH ?= $(shell go env GOARCH)

.PHONY: test
test:
	if [ -n "$(RUN)" ]; then \
		go test $(TEST_BUILDFLAGS) ./pkg/... ./cmd/... -coverprofile cover.out -run $(RUN); \
	else \
		go test $(TEST_BUILDFLAGS) ./pkg/... ./cmd/... -coverprofile cover.out; \
	fi

.PHONY: pact-consumer
pact-consumer:
	mkdir -p pacts/consumer && ( rm -rf pacts/consumer/* || : )
	go test $(TEST_BUILDFLAGS) -v ./contracts/... || true
	if [ "${PACT_PUBLISH_CONTRACT}" = "true" ]; then \
		pact-broker publish ./pacts/consumer \
			--auto-detect-version-properties \
			--consumer-app-version ${GIT_TAG} || true; \
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
kots:
	mkdir -p web/dist
	touch web/dist/README.md
	go build ${LDFLAGS} -o bin/kots $(BUILDFLAGS) github.com/replicatedhq/kots/cmd/kots

.PHONY: build
build:
	mkdir -p web/dist
	touch web/dist/README.md
	go build ${LDFLAGS} ${GCFLAGS} -v -o bin/kotsadm $(BUILDFLAGS) ./cmd/kotsadm

.PHONY: run
run:
	./bin/kotsadm api

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

.PHONY: dev
dev:
	@dev/scripts/dev.sh

.PHONY: %-up
%-up:
	@dev/scripts/up.sh $*

.PHONY: %-down
%-down:
	@dev/scripts/down.sh $*

.PHONY: %-up-ec
%-up-ec:
	@dev/scripts/up-ec.sh $*

.PHONY: %-down-ec
%-down-ec:
	@dev/scripts/down-ec.sh $*

.PHONY: reset
reset:
	kubectl delete -R -f dev/manifests --ignore-not-found

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
build-ttl.sh: web kots build
	docker build --platform $(OS)/$(ARCH) -f dev/dockerfiles/kotsadm/Dockerfile.ttlsh -t ttl.sh/${CURRENT_USER}/kotsadm:24h .
	docker push ttl.sh/${CURRENT_USER}/kotsadm:24h

.PHONY: all-ttl.sh
all-ttl.sh: build-ttl.sh
	source .image.env && \
		IMAGE=ttl.sh/${CURRENT_USER}/kotsadm-migrations:24h \
		DOCKER_BUILD_ARGS="--platform $(OS)/$(ARCH)" \
		make -C migrations build_schema

	docker pull --platform $(OS)/$(ARCH) kotsadm/minio:${MINIO_TAG}
	docker tag kotsadm/minio:${MINIO_TAG} ttl.sh/${CURRENT_USER}/minio:${MINIO_TAG}
	docker push ttl.sh/${CURRENT_USER}/minio:${MINIO_TAG}

	docker pull --platform $(OS)/$(ARCH) kotsadm/rqlite:${RQLITE_TAG}
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
