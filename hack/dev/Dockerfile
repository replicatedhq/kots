FROM golang:1.14 as deps

RUN wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - \
  && echo "deb http://apt.postgresql.org/pub/repos/apt/ buster-pgdg main" > /etc/apt/sources.list.d/PostgreSQL.list \
  && apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client-10 python-pip \
  && pip install s3cmd \
  && rm -rf /var/lib/apt/lists/*


ENV PROJECTPATH=/go/src/github.com/replicatedhq/kots/kotsadm
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
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_16_VERSION}"

ENV KUBECTL_1_17_VERSION=v1.17.13
ENV KUBECTL_1_17_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_17_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_17_SHA256SUM=25824bf20d8d9501e9b6cabdd6bd25ec10cc649d4e3155dba86037f57bba842e
RUN curl -fsSLO "${KUBECTL_1_17_URL}" \
	&& echo "${KUBECTL_1_17_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_17_VERSION}"

ENV KUBECTL_1_18_VERSION=v1.18.10
ENV KUBECTL_1_18_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_18_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_18_SHA256SUM=b25c445cb36ab168de590c13b50bced4e7877ed15ca899a261053b4ba3ba1456
RUN curl -fsSLO "${KUBECTL_1_18_URL}" \
	&& echo "${KUBECTL_1_18_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_18_VERSION}"

ENV KUBECTL_1_19_VERSION=v1.19.3
ENV KUBECTL_1_19_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_19_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_19_SHA256SUM=84eeb8237448e4f431fef0f0ec0ba8b07558d8e52d5a7e89b4ae64dadcffbe66
RUN curl -fsSLO "${KUBECTL_1_19_URL}" \
	&& echo "${KUBECTL_1_19_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_19_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_1_19_VERSION}" /usr/local/bin/kubectl

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
ADD ./bin ./bin

ENTRYPOINT ["./bin", "api"]
