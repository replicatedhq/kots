# syntax=docker/dockerfile:1.3
FROM golang:1.17

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots/kurl_proxy
WORKDIR $PROJECTPATH
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

ENTRYPOINT ["./bin/kurl_proxy"]