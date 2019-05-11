
.PHONY: test
test:
	cd web && make test
	cd api && make test
