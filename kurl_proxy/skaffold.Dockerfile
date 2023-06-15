FROM golang:1.20-buster

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots/kurl_proxy
WORKDIR $PROJECTPATH
ADD Makefile ./
ADD Makefile.build ./
ADD go.mod ./
ADD go.sum ./
ADD cmd ./cmd

RUN make build

ADD assets /assets

ENTRYPOINT ["./bin/kurl_proxy"]
