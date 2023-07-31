# syntax=docker/dockerfile:1.3
FROM golang:1.20.5

EXPOSE 2345

RUN apt update && apt install --no-install-recommends gnupg2 curl -y \
  && apt update && apt install -y --no-install-recommends ca-certificates \
  && apt update && apt-get install -y --no-install-recommends pipx \
	&& pipx install s3cmd && pipx ensurepath \
  && rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/local/bin:$PATH"

# KOTS can be configured to use a specific version of kubectl by setting kubectlVersion in the
# kots.io/v1beta1.Application spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all kubectl binaries in the KOTS_KUBECTL_BIN_DIR directory for use by KOTS.

ENV KOTS_KUBECTL_BIN_DIR=/usr/local/bin

# Install Kubectl 1.14
ENV KUBECTL_1_14_VERSION=v1.14.10
ENV KUBECTL_1_14_URL=https://dl.k8s.io/release/${KUBECTL_1_14_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_14_SHA256SUM=7729c6612bec76badc7926a79b26e0d9b06cc312af46dbb80ea7416d1fce0b36
RUN curl -fsSLO "${KUBECTL_1_14_URL}" \
	&& echo "${KUBECTL_1_14_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.14"

# Install Kubectl 1.16
ENV KUBECTL_1_16_VERSION=v1.16.15
ENV KUBECTL_1_16_URL=https://dl.k8s.io/release/${KUBECTL_1_16_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_16_SHA256SUM=e8913069293156ddf55f243814a22d2384fc18b165efb6200606fdeaad146605
RUN curl -fsSLO "${KUBECTL_1_16_URL}" \
	&& echo "${KUBECTL_1_16_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.16"

# Install Kubectl 1.17
ENV KUBECTL_1_17_VERSION=v1.17.17
ENV KUBECTL_1_17_URL=https://dl.k8s.io/release/${KUBECTL_1_17_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_17_SHA256SUM=8329fac94c66bf7a475b630972a8c0b036bab1f28a5584115e8dd26483de8349
RUN curl -fsSLO "${KUBECTL_1_17_URL}" \
	&& echo "${KUBECTL_1_17_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.17"

# Install Kubectl 1.18
ENV KUBECTL_1_18_VERSION=v1.18.20
ENV KUBECTL_1_18_URL=https://dl.k8s.io/release/${KUBECTL_1_18_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_18_SHA256SUM=66a9bb8e9843050340844ca6e72e67632b75b9ebb651559c49db22f35450ed2f
RUN curl -fsSLO "${KUBECTL_1_18_URL}" \
	&& echo "${KUBECTL_1_18_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.18"

# Install Kubectl 1.19
ENV KUBECTL_1_19_VERSION=v1.19.16
ENV KUBECTL_1_19_URL=https://dl.k8s.io/release/${KUBECTL_1_19_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_19_SHA256SUM=6b9d9315877c624097630ac3c9a13f1f7603be39764001da7a080162f85cbc7e
RUN curl -fsSLO "${KUBECTL_1_19_URL}" \
	&& echo "${KUBECTL_1_19_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.19"

# Install Kubectl 1.20
ENV KUBECTL_1_20_VERSION=v1.20.15
ENV KUBECTL_1_20_URL=https://dl.k8s.io/release/${KUBECTL_1_20_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_20_SHA256SUM=d283552d3ef3b0fd47c08953414e1e73897a1b3f88c8a520bb2e7de4e37e96f3
RUN curl -fsSLO "${KUBECTL_1_20_URL}" \
	&& echo "${KUBECTL_1_20_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.20"

# Install Kubectl 1.21
ENV KUBECTL_1_21_VERSION=v1.21.9
ENV KUBECTL_1_21_URL=https://dl.k8s.io/release/${KUBECTL_1_21_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_21_SHA256SUM=195d5387f2a6ca7b8ab5c2134b4b6cc27f29372f54b771947ba7c18ee983fbe6
RUN curl -fsSLO "${KUBECTL_1_21_URL}" \
	&& echo "${KUBECTL_1_21_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.21"

# Install Kubectl 1.22
ENV KUBECTL_1_22_VERSION=v1.22.6
ENV KUBECTL_1_22_URL=https://dl.k8s.io/release/${KUBECTL_1_22_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_22_SHA256SUM=1ab07643807a45e2917072f7ba5f11140b40f19675981b199b810552d6af5c53
RUN curl -fsSLO "${KUBECTL_1_22_URL}" \
	&& echo "${KUBECTL_1_22_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.22"

# Install Kubectl 1.23
ENV KUBECTL_1_23_VERSION=v1.23.3
ENV KUBECTL_1_23_URL=https://dl.k8s.io/release/${KUBECTL_1_23_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_23_SHA256SUM=d7da739e4977657a3b3c84962df49493e36b09cc66381a5e36029206dd1e01d0
RUN curl -fsSLO "${KUBECTL_1_23_URL}" \
	&& echo "${KUBECTL_1_23_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.23" \
	&& ln -s "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.23" "${KOTS_KUBECTL_BIN_DIR}/kubectl"


ENV KOTS_KUSTOMIZE_BIN_DIR=/usr/local/bin

# KOTS can be configured to use a specific version of kustomize by setting kustomizeVersion in the
# kots.io/v1beta1.Application spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all kustomize binaries in the KOTS_KUSTOMIZE_BIN_DIR directory for use by KOTS.
# CURRENNTLY ONLY ONE VERSION IS SHIPPED BELOW

# Install kustomize 5
ENV KUSTOMIZE5_VERSION=5.0.1
ENV KUSTOMIZE5_URL=https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE5_VERSION}/kustomize_v${KUSTOMIZE5_VERSION}_linux_amd64.tar.gz
ENV KUSTOMIZE5_SHA256SUM=dca623b36aef84fbdf28f79d02e9b3705ff641424ac1f872d5420dadb12fb78d
RUN curl -fsSL -o kustomize.tar.gz "${KUSTOMIZE5_URL}" \
  && echo "${KUSTOMIZE5_SHA256SUM} kustomize.tar.gz" | sha256sum -c - \
  && tar -xzvf kustomize.tar.gz \
  && rm kustomize.tar.gz \
  && chmod a+x kustomize \
  && mv kustomize "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize${KUSTOMIZE5_VERSION}" \
  && ln -s "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize${KUSTOMIZE5_VERSION}" "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize5" \
  && ln -s "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize5" "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize"

# KOTS can be configured to use a specific version of helm by setting helmVersion in the
# kots.io/v1beta1.HelmChart spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all helm binaries in the KOTS_HELM_BIN_DIR directory for use by KOTS.

ENV KOTS_HELM_BIN_DIR=/usr/local/bin

# Install helm v3
ENV HELM3_VERSION=3.12.2
ENV HELM3_URL=https://get.helm.sh/helm-v${HELM3_VERSION}-linux-amd64.tar.gz
ENV HELM3_SHA256SUM=2b6efaa009891d3703869f4be80ab86faa33fa83d9d5ff2f6492a8aebe97b219
RUN cd /tmp && curl -fsSL -o helm.tar.gz "${HELM3_URL}" \
  && echo "${HELM3_SHA256SUM} helm.tar.gz" | sha256sum -c - \
  && tar -xzvf helm.tar.gz \
  && chmod a+x linux-amd64/helm \
  && mv linux-amd64/helm "${KOTS_HELM_BIN_DIR}/helm${HELM3_VERSION}" \
  && ln -s "${KOTS_HELM_BIN_DIR}/helm${HELM3_VERSION}" "${KOTS_HELM_BIN_DIR}/helm3" \
  && ln -s "${KOTS_HELM_BIN_DIR}/helm3" "${KOTS_HELM_BIN_DIR}/helm" \
  && rm -rf helm.tar.gz linux-amd64

ENV DEBUG_KOTSADM=1
ENV GOCACHE "/.cache/gocache/"
ENV GOMODCACHE "/.cache/gomodcache/"
ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots

WORKDIR $PROJECTPATH

RUN --mount=target=$GOMODCACHE,id=gomodcache,type=cache \
    --mount=target=$GOCACHE,id=gocache,type=cache \
    go install github.com/go-delve/delve/cmd/dlv@v1.9.0 

## Get deps
COPY go.mod go.sum ./
RUN --mount=target=$GOMODCACHE,id=kotsadm-gomodcache,type=cache go mod download


## Now add the project and compile
COPY . .
RUN --mount=target=$GOCACHE,id=kotsadm-gocache,type=cache \
    --mount=target=$GOMODCACHE,id=kotsadm-gomodcache,type=cache \
    make build kots

VOLUME [ "$GOCACHE", "$GOMODCACHE" ]

CMD ["make", "run"]
