SHELL:=/bin/bash
PROJECT_NAME ?= kotsadm-migrations
POSTGRES_10_TAG ?= 10.22-alpine
POSTGRES_14_TAG ?= 14.5-alpine
SCHEMAHERO_TAG ?= 0.13.2

.PHONY: schema-alpha
schema-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
schema-alpha: build_schema

.PHONY: schema-release
schema-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
schema-release: NIGHTLY_IMAGE = kotsadm/${PROJECT_NAME}:v0.0.0-nightly
schema-release: build_schema
	docker tag ${IMAGE} ${NIGHTLY_IMAGE}
	docker push ${NIGHTLY_IMAGE}
	mkdir -p bin/docker-archive/${PROJECT_NAME}
	skopeo copy docker-daemon:kotsadm/${PROJECT_NAME}:${GIT_TAG} docker-archive:bin/docker-archive/${PROJECT_NAME}/${GIT_TAG}
	mkdir -p bin/docker-archive/postgres
	skopeo copy docker://postgres:${POSTGRES_10_TAG} docker-archive:bin/docker-archive/postgres/${POSTGRES_10_TAG}
	skopeo copy docker://postgres:${POSTGRES_14_TAG} docker-archive:bin/docker-archive/postgres/${POSTGRES_14_TAG}

build_schema:
	docker build --pull --build-arg SCHEMAHERO_TAG=${SCHEMAHERO_TAG} -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
