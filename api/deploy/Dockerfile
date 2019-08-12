FROM node:8 AS builder
# Install Kubectl
ENV KUBECTL_VERSION=v1.11.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=d16a4e7bfe0033ea5f56f8d11e74f7a2dec5ff8832a046a643c8355b79b4ba5c
RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl
WORKDIR /src
ADD . /src
RUN make deps build pkg


FROM debian:stretch-slim
EXPOSE 3000 9229
WORKDIR /src
COPY --from=builder /src/bin/kotsadm-api /src/kotsadm-api
COPY --from=builder /src/bin/shipcloudctl /src/shipcloudctl
COPY --from=builder /src/node_modules/bcrypt/lib/binding/bcrypt_lib.node /src/node_modules/bcrypt/lib/binding/bcrypt_lib.node

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# Install Kubectl
ENV KUBECTL_VERSION=v1.11.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=d16a4e7bfe0033ea5f56f8d11e74f7a2dec5ff8832a046a643c8355b79b4ba5c
RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}
CMD ["/src/kotsadm-api"]
