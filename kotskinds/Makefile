include ../Makefile.build.mk

.PHONY: generate
generate: controller-gen client-gen
	$(CONTROLLER_GEN) \
		object:headerFile=./hack/boilerplate.go.txt \
		paths=./apis/...
	$(CONTROLLER_GEN) \
		crd \
		+output:dir=./config/crds \
		paths=./apis/kots/v1beta1/...
	$(CLIENT_GEN) \
		--output-package=github.com/replicatedhq/kots/kotskinds/client \
		--clientset-name kotsclientset \
		--input-base github.com/replicatedhq/kots/kotskinds/apis \
		--input kots/v1beta1 \
		-h ./hack/boilerplate.go.txt


.PHONY: openapischema
openapischema: controller-gen
	$(CONTROLLER_GEN) crd +output:dir=./config/crds  paths=./apis/kots/v1beta1

.PHONY: schemas
schemas: fmt generate
	go build ${LDFLAGS} -o bin/schemagen ./schemagen
	./bin/schemagen --output-dir ./schemas

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: controller-gen
controller-gen:
ifeq (, $(shell which controller-gen))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

.PHONY: client-gen
client-gen:
ifeq (, $(shell which client-gen))
	go install k8s.io/code-generator/cmd/client-gen@v0.20.4
CLIENT_GEN=$(shell go env GOPATH)/bin/client-gen
else
CLIENT_GEN=$(shell which client-gen)
endif
