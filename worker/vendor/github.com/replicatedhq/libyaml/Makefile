.PHONY: test docker shell deps

test:
	go test -v -covermode=count .

docker:
	docker build -t libyaml .

shell: docker
	docker run --rm -it --name libyaml \
	  -v "`pwd`:/go/src/github.com/replicatedhq/libyaml" \
	  libyaml

deps:
	go get -t .

install:
	go install .
