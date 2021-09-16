SHELL:=/bin/bash
PROJECT_NAME ?= kotsadm-migrations

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
	skopeo copy docker://postgres:10.17-alpine docker-archive:bin/docker-archive/postgres/10.17-alpine
	skopeo copy docker://postgres:10.17 docker-archive:bin/docker-archive/postgres/10.17

build_schema:
	docker build --pull -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
