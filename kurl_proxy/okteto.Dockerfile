# syntax=docker/dockerfile:1.3
FROM golang:1.22-bookworm

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots/kurl_proxy
WORKDIR $PROJECTPATH
COPY go.mod go.sum ./
RUN go mod download

COPY Makefile Makefile.build ./
COPY cmd cmd

RUN make build

COPY assets assets

ENTRYPOINT ["./bin/kurl_proxy"]
