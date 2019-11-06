FROM debian:stretch-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

# Install Kubectl 1.14
ENV LEGACY_KUBECTL_VERSION=v1.14.7
ENV LEGACY_KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${LEGACY_KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV LEGACY_KUBECTL_SHA256SUM=153fdf0ab8892b4c5e736b3f7c30a89cd4f1753c4d035ec8d771904b17ad181d
RUN curl -fsSLO "${LEGACY_KUBECTL_URL}" \
	&& echo "${LEGACY_KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${LEGACY_KUBECTL_VERSION}"

# Install Kubectl 1.16
ENV KUBECTL_VERSION=v1.16.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=69cfb3eeaa0b77cc4923428855acdfc9ca9786544eeaff9c21913be830869d29
RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

# Setup user
RUN useradd -c 'kotsadm-operator user' -m -d /home/kotsadm-operator -s /bin/bash -u 1001 kotsadm-operator
USER kotsadm-operator
ENV HOME /home/kotsadm-operator

# Install krew
ADD ./deploy/install-krew.sh /install-krew.sh
RUN /install-krew.sh
ENV PATH="$HOME/.krew/bin:$PATH"

# Install our plugins
ENV TROUBLESHOOT_VERSION=0.9.13
RUN kubectl krew install preflight
RUN kubectl krew install support-bundle

## This is used when we aren't on a release of preflight
# RUN curl -L "https://github.com/replicatedhq/troubleshoot/releases/download/v0.9.8/preflight_0.9.8_linux_amd64-0.9.8.tar.gz" > /tmp/preflight.tar.gz && \
#   cd /tmp && tar xzvf preflight.tar.gz && \
#   mv /tmp/preflight /usr/local/bin/preflight

COPY ./bin/kotsadm-operator /kotsadm-operator
USER root
RUN chmod a+x /kotsadm-operator
USER kotsadm-operator

EXPOSE 3000
# ARG version=unknown
# ENV VERSION=${version}
ENTRYPOINT ["/kotsadm-operator"]
