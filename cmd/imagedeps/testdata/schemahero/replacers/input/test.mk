CURRENT_USER := $(shell id -u -n)
SCHEMAHERO_TAG ?= 0.13.1

.PHONY: test
test:
	go test ./pkg/...
	