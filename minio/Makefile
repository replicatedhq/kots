SHELL := /bin/bash
PROJECT_NAME ?= minio

.PHONY: publish
publish: IMAGE = kotsadm/${PROJECT_NAME}:${IMAGE_TAG}
publish:
	docker build -f Dockerfile -t ${IMAGE} .
	docker push ${IMAGE}

.PHONY: publish-latest
publish-latest: IMAGE_TAG = latest
publish-latest: publish

.PHONY: publish-release
publish-release: IMAGE_TAG = ${GIT_TAG}
publish-release: publish