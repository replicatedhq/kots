CURRENT_USER := $(shell id -u -n)
SCHEMAHERO_TAG ?= 0.13.2

.PHONY: test
test:
	go test ./pkg/...
	