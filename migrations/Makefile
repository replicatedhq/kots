SHELL:=/bin/bash
SCHEMAHERO_TAG ?= 0.17.5

build_schema:
	docker build --pull --build-arg SCHEMAHERO_TAG=${SCHEMAHERO_TAG} -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
