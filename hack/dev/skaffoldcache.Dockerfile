FROM golang:1.20 as deps

RUN go install github.com/go-delve/delve/cmd/dlv@v1.7.2

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots
WORKDIR $PROJECTPATH
RUN mkdir -p web/dist && touch web/dist/README.md
COPY Makefile ./
COPY Makefile.build.mk ./
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY pkg ./pkg
COPY web/webcontent.go ./web/webcontent.go

RUN make build kots
