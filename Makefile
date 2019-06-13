SHELL := /bin/bash

.PHONY: cache
cache:
	make -C api build-cache
	make -C worker build-cache
	make -C ship-operator build-cache
	make -C ship-operator-tools build-cache
	make -C web build-cache

.PHONY: test
test:
	make -C web test
	make -C ship-cd test-pact

	make -C migrations/fixtures schema-fixtures build run
	cd migrations && docker build -t replicated/ship-cluster-fixtures:local -f ./fixtures/deploy/Dockerfile ./fixtures

	mkdir -p api/pacts
	cp web/pacts/ship-cluster-ui-ship-cluster-api.json api/pacts/
	cp ship-cd/pacts/ship-cd-ship-cluster-api.json api/pacts/
	make -C api test

	@echo All contract tests have passed.

.PHONY: bitbucket-server
bitbucket-server:
	docker volume create --name bitbucketVolume
	@-docker stop bitbucket > /dev/null 2>&1 || :
	@-docker rm -f bitbucket > /dev/null 2>&1 || :
	docker run \
		-v bitbucketVolume:/var/atlassian/application-data/bitbucket \
		--name="bitbucket" \
		-d \
		-p 7990:7990 \
		-p 7999:7999 \
		atlassian/bitbucket-server:4.12
	@echo "A BitBucket server is starting on http://localhost:7990. You'll need to install an eval license".

.PHONY: reset-ships
reset-ships:
	kubectl delete ns `kubectl get ns | grep shipwatch- | awk '{print $1}'` || true
	kubectl delete ns `kubectl get ns | grep shipedit- | awk '{print $1}'` || true
	kubectl delete ns `kubectl get ns | grep shipupdate- | awk '{print $1}'` || true
	kubectl delete ns `kubectl get ns | grep shipinit- | awk '{print $1}'` || terue
