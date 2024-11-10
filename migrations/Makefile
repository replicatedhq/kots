SHELL:=/bin/bash
SCHEMAHERO_TAG ?= 0.17.12
DOCKER_BUILD_ARGS ?=

build_schema:
	docker build --pull --build-arg SCHEMAHERO_TAG=${SCHEMAHERO_TAG} ${DOCKER_BUILD_ARGS} -f dev/Dockerfile.ttlsh -t ${IMAGE} .
	docker push ${IMAGE}
