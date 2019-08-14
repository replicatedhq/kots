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

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /src
ADD . /src
RUN make deps build

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}

CMD ["node", "/src/build/server/index.js"]
