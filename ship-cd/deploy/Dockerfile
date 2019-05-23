# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/replicatedhq/ship-cd
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager github.com/replicatedhq/ship-cd/cmd/manager


# Copy the controller-manager into a thin image
FROM ubuntu:latest
WORKDIR /
COPY --from=builder /go/src/github.com/replicatedhq/ship-cd/manager .

RUN apt-get update && apt-get install -y ca-certificates curl && rm -rf /var/lib/apt/lists/*

# Install Kubectl
ENV KUBECTL_VERSION=v1.11.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=d16a4e7bfe0033ea5f56f8d11e74f7a2dec5ff8832a046a643c8355b79b4ba5c
RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

ENTRYPOINT ["/manager"]
