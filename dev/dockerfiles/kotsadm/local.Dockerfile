FROM golang:1.22

RUN go install github.com/go-delve/delve/cmd/dlv@v1.22.1

ENV GO111MODULE=on
ENV PATH="/usr/local/bin:$PATH"

RUN apt-get update && apt-get install -y --no-install-recommends s3cmd

# Install Kubectl 1.29
ENV KUBECTL_1_29_VERSION=v1.29.0
ENV KUBECTL_1_29_URL=https://dl.k8s.io/release/${KUBECTL_1_29_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_29_SHA256SUM=0e03ab096163f61ab610b33f37f55709d3af8e16e4dcc1eb682882ef80f96fd5
RUN curl -fsSLO "${KUBECTL_1_29_URL}" \
  && echo "${KUBECTL_1_29_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl /usr/local/bin/kubectl

# Install kustomize 5
ENV KUSTOMIZE5_VERSION=5.1.1
ENV KUSTOMIZE5_URL=https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE5_VERSION}/kustomize_v${KUSTOMIZE5_VERSION}_linux_amd64.tar.gz
ENV KUSTOMIZE5_SHA256SUM=3b30477a7ff4fb6547fa77d8117e66d995c2bdd526de0dafbf8b7bcb9556c85d
RUN curl -fsSL -o kustomize.tar.gz "${KUSTOMIZE5_URL}" \
  && echo "${KUSTOMIZE5_SHA256SUM} kustomize.tar.gz" | sha256sum -c - \
  && tar -xzvf kustomize.tar.gz \
  && rm kustomize.tar.gz \
  && chmod a+x kustomize \
  && mv kustomize /usr/local/bin/kustomize

# Install helm v3
ENV HELM3_VERSION=3.13.2
ENV HELM3_URL=https://get.helm.sh/helm-v${HELM3_VERSION}-linux-amd64.tar.gz
ENV HELM3_SHA256SUM=55a8e6dce87a1e52c61e0ce7a89bf85b38725ba3e8deb51d4a08ade8a2c70b2d
RUN cd /tmp && curl -fsSL -o helm.tar.gz "${HELM3_URL}" \
  && echo "${HELM3_SHA256SUM} helm.tar.gz" | sha256sum -c - \
  && tar -xzvf helm.tar.gz \
  && chmod a+x linux-amd64/helm \
  && mv linux-amd64/helm /usr/local/bin/helm \
  && rm -rf helm.tar.gz linux-amd64

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots
WORKDIR $PROJECTPATH
COPY Makefile ./
COPY Makefile.build.mk ./
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY pkg ./pkg
COPY web/webcontent.go ./web/webcontent.go

RUN make build kots

RUN mv ./bin/kotsadm /kotsadm
RUN mv ./bin/kots /kots

ARG DEBUG_KOTSADM=0
ENV DEBUG_KOTSADM=${DEBUG_KOTSADM}

ADD dev/dockerfiles/kotsadm/entrypoint.sh .
ENTRYPOINT [ "./entrypoint.sh"]
