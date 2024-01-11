CURRENT_USER := $(if $(GITHUB_USER),$(GITHUB_USER),$(shell id -u -n))
SCHEMAHERO_TAG ?= 0.13.2

.PHONY: test
test:
	go test ./pkg/...
	