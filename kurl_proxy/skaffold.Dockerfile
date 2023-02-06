FROM golang:1.19-bullseye

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
