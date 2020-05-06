FROM golang:1.14 as deps

ENV PROJECTPATH=/go/src/github.com/replicatedhq/kotsadm
WORKDIR $PROJECTPATH
ADD Makefile $PROJECTPATH/
# ADD go.mod $PROJECTPATH/
# ADD go.sum $PROJECTPATH/

ENV GO111MODULE=on

# Install Kubectl 1.14
ENV KUBECTL_1_14_VERSION=v1.14.9
ENV KUBECTL_1_14_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_14_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_14_SHA256SUM=d2a31e87c5f6deced4ba8899f9c465e54822f0cd146f32ea83cb1daafa5d9c4f
RUN curl -fsSLO "${KUBECTL_1_14_URL}" \
	&& echo "${KUBECTL_1_14_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_14_VERSION}"

# Install Kubectl 1.16
ENV KUBECTL_1_16_VERSION=v1.16.3
ENV KUBECTL_1_16_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_16_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_16_SHA256SUM=cded1b46405741575f31024b757fd967645e815bb0ab1c5f5fcd029f25cc0f2d
RUN curl -fsSLO "${KUBECTL_1_16_URL}" \
	&& echo "${KUBECTL_1_16_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_16_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_1_16_VERSION}" /usr/local/bin/kubectl

RUN curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/v2.0.3/kustomize_2.0.3_linux_amd64" > /tmp/kustomize && \
  chmod a+x /tmp/kustomize && \
  mv /tmp/kustomize "/usr/local/bin/kustomize2.0.3"

# Install kustomize 3
RUN curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz" > /tmp/kustomize.tar.gz && \
  tar -xzvf /tmp/kustomize.tar.gz && \
  rm /tmp/kustomize.tar.gz && \
  chmod a+x kustomize && \
  mv kustomize "/usr/local/bin/kustomize3.5.4"
	
# Install krew
ADD ./deploy/install-krew.sh /install-krew.sh
RUN /install-krew.sh
ENV PATH="/root/.krew/bin:$PATH"

# Install our plugins
RUN kubectl krew install preflight
RUN kubectl krew install support-bundle

# ADD cmd $PROJECTPATH/cmd
# ADD pkg $PROJECTPATH/pkg
# ADD web/dist $PROJECTPATH/web/dist
ADD ./bin/kotsadm ./bin/kotsadm

ENTRYPOINT ["./bin/kotsadm", "api"]
