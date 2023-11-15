# syntax=docker/dockerfile:1.3
FROM golang:1.21-bookworm as builder

EXPOSE 2345

ENV GOCACHE "/.cache/gocache/"
ENV GOMODCACHE "/.cache/gomodcache/"
ENV DEBUG_KOTSADM=1

RUN apt-get update && apt-get install -y --no-install-recommends gnupg2 s3cmd ca-certificates \
  && rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/local/bin:$PATH"

# KOTS can be configured to use a specific version of kubectl by setting kubectlVersion in the
# kots.io/v1beta1.Application spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all kubectl binaries in the KOTS_KUBECTL_BIN_DIR directory for use by KOTS.

ENV KOTS_KUBECTL_BIN_DIR=/usr/local/bin

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
ENV KUBECTL_1_21_VERSION=v1.21.14
ENV KUBECTL_1_21_URL=https://dl.k8s.io/release/${KUBECTL_1_21_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_21_SHA256SUM=0c1682493c2abd7bc5fe4ddcdb0b6e5d417aa7e067994ffeca964163a988c6ee
RUN curl -fsSLO "${KUBECTL_1_21_URL}" \
  && echo "${KUBECTL_1_21_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.21"

# Install Kubectl 1.22
ENV KUBECTL_1_22_VERSION=v1.22.17
ENV KUBECTL_1_22_URL=https://dl.k8s.io/release/${KUBECTL_1_22_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_22_SHA256SUM=7506a0ae7a59b35089853e1da2b0b9ac0258c5309ea3d165c3412904a9051d48
RUN curl -fsSLO "${KUBECTL_1_22_URL}" \
  && echo "${KUBECTL_1_22_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.22"

# Install Kubectl 1.23
ENV KUBECTL_1_23_VERSION=v1.23.17
ENV KUBECTL_1_23_URL=https://dl.k8s.io/release/${KUBECTL_1_23_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_23_SHA256SUM=f09f7338b5a677f17a9443796c648d2b80feaec9d6a094ab79a77c8a01fde941
RUN curl -fsSLO "${KUBECTL_1_23_URL}" \
  && echo "${KUBECTL_1_23_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.23"

# Install Kubectl 1.24
ENV KUBECTL_1_24_VERSION=v1.24.17
ENV KUBECTL_1_24_URL=https://dl.k8s.io/release/${KUBECTL_1_24_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_24_SHA256SUM=3e9588e3326c7110a163103fc3ea101bb0e85f4d6fd228cf928fa9a2a20594d5
RUN curl -fsSLO "${KUBECTL_1_24_URL}" \
  && echo "${KUBECTL_1_24_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.24"

# Install Kubectl 1.25
ENV KUBECTL_1_25_VERSION=v1.25.15
ENV KUBECTL_1_25_URL=https://dl.k8s.io/release/${KUBECTL_1_25_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_25_SHA256SUM=6428297af0b06d1bb87601258fb61c13d82bf3187b2329b5f38b6f0fec5be575
RUN curl -fsSLO "${KUBECTL_1_25_URL}" \
  && echo "${KUBECTL_1_25_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.25"

# Install Kubectl 1.26
ENV KUBECTL_1_26_VERSION=v1.26.10
ENV KUBECTL_1_26_URL=https://dl.k8s.io/release/${KUBECTL_1_26_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_26_SHA256SUM=93ad44b4072669237247bfbc171be816f08e7e9e4260418d2cfdd0da1704ae86
RUN curl -fsSLO "${KUBECTL_1_26_URL}" \
  && echo "${KUBECTL_1_26_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.26"

# Install Kubectl 1.27
ENV KUBECTL_1_27_VERSION=v1.27.7
ENV KUBECTL_1_27_URL=https://dl.k8s.io/release/${KUBECTL_1_27_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_27_SHA256SUM=e5fe510ba6f421958358d3d43b3f0b04c2957d4bc3bb24cf541719af61a06d79
RUN curl -fsSLO "${KUBECTL_1_27_URL}" \
  && echo "${KUBECTL_1_27_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.27"

# Install Kubectl 1.28
ENV KUBECTL_1_28_VERSION=v1.28.3
ENV KUBECTL_1_28_URL=https://dl.k8s.io/release/${KUBECTL_1_28_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_28_SHA256SUM=0c680c90892c43e5ce708e918821f92445d1d244f9b3d7513023bcae9a6246d1
RUN curl -fsSLO "${KUBECTL_1_28_URL}" \
  && echo "${KUBECTL_1_28_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.28" \
  && ln -s "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.28" "${KOTS_KUBECTL_BIN_DIR}/kubectl"


ENV KOTS_KUSTOMIZE_BIN_DIR=/usr/local/bin

# KOTS can be configured to use a specific version of kustomize by setting kustomizeVersion in the
# kots.io/v1beta1.Application spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all kustomize binaries in the KOTS_KUSTOMIZE_BIN_DIR directory for use by KOTS.
# CURRENNTLY ONLY ONE VERSION IS SHIPPED BELOW

# Install kustomize 5
ENV KUSTOMIZE5_VERSION=5.1.1
ENV KUSTOMIZE5_URL=https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE5_VERSION}/kustomize_v${KUSTOMIZE5_VERSION}_linux_amd64.tar.gz
ENV KUSTOMIZE5_SHA256SUM=3b30477a7ff4fb6547fa77d8117e66d995c2bdd526de0dafbf8b7bcb9556c85d
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
ENV HELM3_VERSION=3.13.2
ENV HELM3_URL=https://get.helm.sh/helm-v${HELM3_VERSION}-linux-amd64.tar.gz
ENV HELM3_SHA256SUM=55a8e6dce87a1e52c61e0ce7a89bf85b38725ba3e8deb51d4a08ade8a2c70b2d
RUN cd /tmp && curl -fsSL -o helm.tar.gz "${HELM3_URL}" \
  && echo "${HELM3_SHA256SUM} helm.tar.gz" | sha256sum -c - \
  && tar -xzvf helm.tar.gz \
  && chmod a+x linux-amd64/helm \
  && mv linux-amd64/helm "${KOTS_HELM_BIN_DIR}/helm${HELM3_VERSION}" \
  && ln -s "${KOTS_HELM_BIN_DIR}/helm${HELM3_VERSION}" "${KOTS_HELM_BIN_DIR}/helm3" \
  && ln -s "${KOTS_HELM_BIN_DIR}/helm3" "${KOTS_HELM_BIN_DIR}/helm" \
  && rm -rf helm.tar.gz linux-amd64

RUN --mount=target=$GOMODCACHE,id=gomodcache,type=cache \
    --mount=target=$GOCACHE,id=gocache,type=cache \
    go install github.com/go-delve/delve/cmd/dlv@v1.8.0

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots
WORKDIR $PROJECTPATH

COPY go.mod go.sum ./
RUN --mount=target=$GOMODCACHE,id=kots-gomodcache,type=cache go mod download

COPY . .

RUN --mount=target=$GOMODCACHE,id=kots-gomodcache,type=cache \
    --mount=target=$GOCACHE,id=kots-gocache,type=cache \
    make build kots && \
    mv ./bin/kotsadm /kotsadm && \
    mv ./bin/kots /kots

RUN --mount=target=/tmp/.cache/gocache,id=kots-gocache,type=cache \
    --mount=target=/tmp/.cache/gomodcache,id=kots-gomodcache,type=cache \
    mkdir -p $GOCACHE \
    && cp -r /tmp/.cache/gocache/* $GOCACHE \
    && mkdir -p $GOMODCACHE \
    && cp -r /tmp/.cache/gomodcache/* $GOMODCACHE

ENTRYPOINT [ "/kotsadm", "api"]
