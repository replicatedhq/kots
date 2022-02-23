FROM schemahero/schemahero:0.13.0-alpha.1 as schemahero

USER root
RUN apt-get install -y build-essential
USER schemahero

ADD tables/ /go/src/github.com/replicatedhq/kots/tables
WORKDIR /go/src/github.com/replicatedhq/kots/tables
