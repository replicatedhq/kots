FROM debian:stretch-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

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

# Setup user
RUN useradd -c 'kotsadm-operator user' -m -d /home/kotsadm-operator -s /bin/bash -u 1001 kotsadm-operator
USER kotsadm-operator
ENV HOME /home/kotsadm-operator

# Install krew
ADD ./deploy/install-krew.sh /install-krew.sh
RUN /install-krew.sh
ENV PATH="$HOME/.krew/bin:$PATH"

COPY ./bin/kotsadm-operator /kotsadm-operator
USER root
RUN chmod a+x /kotsadm-operator
USER kotsadm-operator

# Install our plugins
USER root
ENV TROUBLESHOOT_VERSION=0.9.31
RUN curl -L "https://github.com/replicatedhq/troubleshoot/releases/download/v${TROUBLESHOOT_VERSION}/preflight_linux_amd64.tar.gz" > /tmp/preflight.tar.gz && \
  cd /tmp && tar xzvf preflight.tar.gz && \
  mv /tmp/preflight /usr/local/bin/kubectl-preflight

RUN curl -L "https://github.com/replicatedhq/troubleshoot/releases/download/v${TROUBLESHOOT_VERSION}/support-bundle_linux_amd64.tar.gz" > /tmp/support-bundle.tar.gz && \
  cd /tmp && tar xzvf support-bundle.tar.gz && \
  mv /tmp/support-bundle /usr/local/bin/kubectl-support_bundle
USER kotsadm-operator

EXPOSE 3000
# ARG version=unknown
# ENV VERSION=${version}
ENTRYPOINT ["/kotsadm-operator"]
