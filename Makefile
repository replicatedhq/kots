SHELL := /bin/bash

.PHONY: test
test:
	make -C web test
	make -C operator test

	make -C migrations/fixtures schema-fixtures build run
	cd migrations && docker build -t kotsadm/kotsadm-fixtures:local -f ./fixtures/deploy/Dockerfile ./fixtures

	mkdir -p api/pacts
	cp web/pacts/kotsadm-web-kotsadm-api.json api/pacts/
	make -C api test

	@echo All contract tests have passed.

.PHONY: kots-local-build
kots-local-build: REGISTRY=registry.somebigbank.com
kots-local-build: NAMESPACE=kotsadm
kots-local-build: TAG=special
kots-local-build:
	cd api; docker build -f deploy/Dockerfile -t ${REGISTRY}/${NAMESPACE}/kotsadm-api:${TAG} . && docker push ${REGISTRY}/${NAMESPACE}/kotsadm-api:${TAG}
	cd web; make build-kotsadm && docker build --build-arg=nginxconf=deploy/kotsadm.conf -f deploy/Dockerfile -t ${REGISTRY}/${NAMESPACE}/kotsadm-web:${TAG} . && docker push ${REGISTRY}/${NAMESPACE}/kotsadm-web:${TAG}
	cd operator; docker build -f deploy/Dockerfile -t ${REGISTRY}/${NAMESPACE}/kotsadm-operator:${TAG} . && docker push ${REGISTRY}/${NAMESPACE}/kotsadm-operator:${TAG}
	docker pull schemahero/schemahero:alpha
	cd migrations && docker build -f deploy/Dockerfile -t ${REGISTRY}/${NAMESPACE}/kotsadm-migrations:${TAG} . && docker push ${REGISTRY}/${NAMESPACE}/kotsadm-migrations:${TAG}


