
export GO111MODULE=on

.PHONY: generate
generate: controller-gen client-gen
	controller-gen \
		object:headerFile=./hack/boilerplate.go.txt \
		paths=./apis/...
	client-gen \
		--output-package=github.com/replicatedhq/kots/kotskinds/client \
		--clientset-name kotsclientset \
		--input-base github.com/replicatedhq/kots/kotskinds/apis \
		--input kots/v1beta1 \
		-h ./hack/boilerplate.go.txt

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.2
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# find or download client-gen
client-gen:
ifeq (, $(shell which client-gen))
	go get k8s.io/code-generator/cmd/client-gen@kubernetes-1.13.5
CLIENT_GEN=$(shell go env GOPATH)/bin/client-gen
else
CLIENT_GEN=$(shell which client-gen)
endif
