include ./Makefile.build
CURRENT_USER := $(shell id -u -n)

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

.PHONY: fmt
fmt:
	go fmt ./pkg/... ./cmd/...

.PHONY: vet
vet:
	go vet ./pkg/... ./cmd/...

.PHONY: test
test: fmt vet pacts
	go test ./pkg/... ./cmd/...

.PHONY: build
build:
	go build ${LDFLAGS} -o bin/kotsadm-operator github.com/replicatedhq/kots/kotsadm/operator/cmd/kotsadm-operator

.PHONY: run
run: bin
	./bin/kotsadm-operator --api-endpoint http://localhost:30000 --token local

.PHONY: build-ttl.sh
build-ttl.sh:
	docker build -f deploy/Dockerfile -t ttl.sh/${CURRENT_USER}/kotsadm-operator:12h .
	docker push ttl.sh/${CURRENT_USER}/kotsadm-operator:12h

.PHONY: build-alpha
build-alpha:
	docker build -f deploy/Dockerfile -t kotsadm/kotsadm-operator:alpha .
	docker push kotsadm/kotsadm-operator:alpha

.PHONY: build-release
build-release:
	docker build -f deploy/Dockerfile -t kotsadm/kotsadm-operator:${GIT_TAG} .
	docker push kotsadm/kotsadm-operator:${GIT_TAG}
	mkdir -p bin/docker-archive/kotsadm-operator
	skopeo copy docker-daemon:kotsadm/kotsadm-operator:${GIT_TAG} docker-archive:bin/docker-archive/kotsadm-operator/${GIT_TAG}
