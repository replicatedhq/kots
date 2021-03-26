SHELL := /bin/bash
PROJECT_NAME ?= postgres

.PHONY: publish
publish: IMAGE = kotsadm/${PROJECT_NAME}:${IMAGE_TAG}
publish:
	docker build -f Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}
	
.PHONY: publish-latest
publish-latest: IMAGE_TAG = alpha
publish-latest: publish

.PHONY: publish-release
publish-release: IMAGE_TAG = ${GIT_TAG}
publish-release: publish
	mkdir -p bin/docker-archive/${PROJECT_NAME}
	skopeo copy docker-daemon:kotsadm/${PROJECT_NAME}:${GIT_TAG} docker-archive:bin/docker-archive/${PROJECT_NAME}/${GIT_TAG}
