CURRENT_USER := $(if $(GITHUB_USER),$(GITHUB_USER),$(shell id -u -n))
SCHEMAHERO_TAG ?= 0.13.1

.PHONY: test
test:
	go test ./pkg/...
	