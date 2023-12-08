SHELL:=/bin/bash
PROJECT_NAME ?= kotsadm-migrations
RQLITE_TAG ?= 8.0.0
SCHEMAHERO_TAG ?= 0.17.0

.PHONY: schema-alpha
schema-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
schema-alpha: build_schema

.PHONY: schema-release
schema-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
schema-release: build_schema
	mkdir -p bin/docker-archive/${PROJECT_NAME}
	skopeo copy docker-daemon:kotsadm/${PROJECT_NAME}:${GIT_TAG} docker-archive:bin/docker-archive/${PROJECT_NAME}/${GIT_TAG}
	mkdir -p bin/docker-archive/rqlite
	skopeo copy docker://rqlite/rqlite:${RQLITE_TAG} docker-archive:bin/docker-archive/rqlite/${RQLITE_TAG}

build_schema:
	docker build --pull --build-arg SCHEMAHERO_TAG=${SCHEMAHERO_TAG} -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
