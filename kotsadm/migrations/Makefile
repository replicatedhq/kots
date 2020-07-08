SHELL:=/bin/bash
PROJECT_NAME ?= kotsadm-migrations

.PHONY: schema-alpha
schema-alpha: IMAGE = kotsadm/${PROJECT_NAME}:alpha
schema-alpha: build_schema

.PHONY: schema-release
schema-release: IMAGE = kotsadm/${PROJECT_NAME}:${GIT_TAG}
schema-release: build_schema
	mkdir -p bin/docker-archive/${PROJECT_NAME}
	skopeo copy docker-daemon:kotsadm/${PROJECT_NAME}:${GIT_TAG} docker-archive:bin/docker-archive/${PROJECT_NAME}/${GIT_TAG}
	mkdir -p bin/docker-archive/postgres
	skopeo copy docker://postgres:10.7 docker-archive:bin/docker-archive/postgres/${GIT_TAG}

.PHONY: build-ttl-ci.sh
build-ttl-ci.sh: IMAGE = ttl.sh/${UUID}/${PROJECT_NAME}:2h
build-ttl-ci.sh: build_schema

build_schema:
	docker pull schemahero/schemahero:0.9.0
	docker build -f deploy/Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
