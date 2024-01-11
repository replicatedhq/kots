
SHELL := /bin/bash

export PATH := $(GOPATH):$(PATH)

GOSRC = $(shell find ../../../../cmd/ ../../../../pkg/ -name "*")
WEBSRC = $(shell find ../../../../web/src/ -name "*") ../../../../web/package.json
KURLPROXYSRC = $(shell find ../../../../kurl_proxy/ -name "*")
MIGRATIOSSRC = $(shell find ../../../../migrations/ -name "*")

.PHONY: all
all: build kurl generate run

.PHONY: build
build: build/kots build/images

build/kots: $(GOSRC)
	${MAKE} -C ../../../../ kots
	@mkdir -p build
	@cp ../../../../bin/kots build/kots

build/images: $(GOSRC) $(WEBSRC) $(KURLPROXYSRC) $(MIGRATIOSSRC) ../../../../.image.env
	${MAKE} -C ../../../../ all-ttl.sh
	${MAKE} -C ../../../../kurl_proxy build-ttl.sh
	@mkdir -p build
	@touch build/images

.PHONY: kurl
kurl: BASE_DIR = kurl
kurl:
	@[ -d $(BASE_DIR) ] || git clone --depth=1 --branch=main git@github.com:replicatedhq/kurl.git $(BASE_DIR)
	@cd $(BASE_DIR) && git pull --depth=1

.PHONY: generate
generate:
	@RQLITE_TAG=$(shell source ../../../../.image.env && echo "$$RQLITE_TAG") \
	DEX_TAG=$(shell source ../../../../.image.env && echo "$$DEX_TAG") \
		${MAKE} _generate
	
_generate: CURRENT_USER = $(if $(GITHUB_USER),$(GITHUB_USER),$(shell id -u -n))
_generate: generate-tag
	@cd ../template && ./generate.sh $(KOTS_VERSION) ttl.sh $(CURRENT_USER) 24h $(shell realpath ../../../../bin/kots)
	@rm -rf kurl/addons/kotsadm/$(KOTS_VERSION)
	@cp -r ../$(KOTS_VERSION) kurl/addons/kotsadm/
	mkdir -p kurl/addons/kotsadm/$(KOTS_VERSION)/assets/
	tar czvf kurl/addons/kotsadm/$(KOTS_VERSION)/assets/kots.tar.gz ../../../../bin/kots

.PHONY: generate-tag
generate-tag:
	$(eval KOTS_VERSION = $(shell echo "$(shell date +'%Y.%-m.%-d')-$(shell git rev-parse --short HEAD)-nightly"))

.PHONY: run
run: build/Manifest
	@cd kurl && npm i
	@cp build/Manifest kurl/scripts/Manifest
	REMOTES=$(REMOTES) NO_BIN=$(NO_BIN) ${MAKE} -C kurl watchrsync

.PHONY: build/Manifest
build/Manifest: build/installer.yaml
	@cd kurl && git checkout scripts/Manifest
	@cat kurl/scripts/Manifest | sed 's|INSTALLER_YAML=|INSTALLER_YAML="|' > build/Manifest
	@cat build/installer.yaml >> build/Manifest
	echo '"' >> build/Manifest

build/installer.yaml: generate-tag
	@mkdir -p build
	cat installer.yaml | sed 's|__KOTSADM_VERSION__|$(KOTS_VERSION)|' > build/installer.yaml
