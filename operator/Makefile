
SHELL := /bin/bash -o pipefail
PROJECT_NAME ?= kotsadm-opetator

.PHONY: publish-pact
publish-pact:
	mkdir -p pacts; buildkite-agent artifact download pacts/* pacts --step test-and-build
	curl \
	--silent --output /dev/null --show-error --fail \
	--user ${PACT_BROKER_USERNAME}:${PACT_BROKER_PASSWORD} \
	-X PUT \
	-H "Content-Type: application/json" \
	-d@pacts/operator-ship-cluster-api.json \
	https://replicated-pact-broker.herokuapp.com/pacts/provider/ship-cluster-api/consumer/operator/version/0.0.1

.PHONY: pacts
pacts:
	go test ./pkg/controller/cluster/pacts

.PHONY: test
test: pacts

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: test
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: bin
bin: fmt vet
	go build -o bin/kotsadm-operator github.com/replicatedhq/ship-cluster/operator/cmd/kotsadm-operator

.PHONY: runs
run: bin
	./bin/kotsadm-operator --api-endpoint http://localhost:30065 --token local
