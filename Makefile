SHELL := /bin/bash

/lib/kots.so:
	curl -L "https://github.com/replicatedhq/kots/releases/download/v0.9.0/kots.so_linux_amd64.tar.gz" > /tmp/kots.tar.gz && \
  cd /tmp && tar xzvf kots.tar.gz && \
  sudo mv /tmp/kots.so /lib/kots.so

.PHONY: test
test: /lib/kots.so
	make -C web test
	make -C operator test
	make -C migrations/fixtures schema-fixtures build run
	cd migrations && docker build -t kotsadm/kotsadm-fixtures:local -f ./fixtures/deploy/Dockerfile ./fixtures

	mkdir -p api/pacts
	cp web/pacts/kotsadm-web-kotsadm-api.json api/pacts/
	make -C api test

	@echo All contract tests have passed.

.PHONY: reset-ships
reset-ships:
	kubectl delete ns `kubectl get ns | grep shipwatch- | awk '{print $1}'` || true
	kubectl delete ns `kubectl get ns | grep shipedit- | awk '{print $1}'` || true
	kubectl delete ns `kubectl get ns | grep shipupdate- | awk '{print $1}'` || true
	kubectl delete ns `kubectl get ns | grep shipinit- | awk '{print $1}'` || true
