SHELL := /bin/bash
PROJECT_NAME ?= minio

.PHONY: publish
publish: IMAGE = kotsadm/${PROJECT_NAME}:${IMAGE_TAG}
publish:
	docker build -f Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}

.PHONY: build-ttl-ci.sh
build-ttl-ci.sh:
	docker build -f Dockerfile -t ttl.sh/${UUID}/${PROJECT_NAME}:2h .
	docker push ttl.sh/${UUID}/${PROJECT_NAME}:2h

.PHONY: publish-latest
publish-latest: IMAGE_TAG = alpha
publish-latest: publish

.PHONY: publish-release
publish-release: IMAGE_TAG = ${GIT_TAG}
publish-release: publish
	mkdir -p bin/docker-archive
	skopeo copy docker-daemon:kotsadm/${PROJECT_NAME}:${GIT_TAG} docker-archive:bin/docker-archive/${PROJECT_NAME}.tar
