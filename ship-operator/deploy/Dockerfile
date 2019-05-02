# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/replicatedhq/ship-operator
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ship-operator github.com/replicatedhq/ship-operator/cmd/ship-operator

# Copy the controller-manager into a thin image
FROM ubuntu:latest
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /go/src/github.com/replicatedhq/ship-operator/ship-operator .
ENTRYPOINT ["./ship-operator"]
