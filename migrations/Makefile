SHELL:=/bin/bash
SCHEMAHERO_TAG ?= 0.17.16
DOCKER_BUILD_ARGS ?=

build_schema:
	docker build --pull --build-arg SCHEMAHERO_TAG=${SCHEMAHERO_TAG} ${DOCKER_BUILD_ARGS} -f ../dev/dockerfiles/kotsadm-migrations/Dockerfile.ttlsh -t ${IMAGE} .
	docker push ${IMAGE}
