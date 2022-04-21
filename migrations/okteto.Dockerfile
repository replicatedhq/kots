# syntax=docker/dockerfile:1.3
FROM schemahero/schemahero:0.13.0-alpha.1 as schemahero

USER root
RUN apt-get update && apt-get install -y build-essential
USER schemahero

WORKDIR /go/src/github.com/replicatedhq/kots/tables
COPY tables/ .