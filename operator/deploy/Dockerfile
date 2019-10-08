FROM debian:stretch-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git \
  && rm -rf /var/lib/apt/lists/*

# Install Kubectl
ENV KUBECTL_VERSION=v1.14.7
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=153fdf0ab8892b4c5e736b3f7c30a89cd4f1753c4d035ec8d771904b17ad181d
RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

# Install krew
ADD ./deploy/install-krew.sh /install-krew.sh
RUN /install-krew.sh
ENV PATH="/root/.krew/bin:$PATH"

# Install our plugins
# RUN kubectl krew install preflight
RUN kubectl krew install support-bundle

## This is used when we aren't on a release of preflight
RUN curl -L "https://github.com/replicatedhq/troubleshoot/releases/download/v0.9.8/preflight_0.9.8_linux_amd64-0.9.8.tar.gz" > /tmp/preflight.tar.gz && \
  cd /tmp && tar xzvf preflight.tar.gz && \
  mv /tmp/preflight /usr/local/bin/preflight

COPY ./bin/kotsadm-operator /kotsadm-operator
RUN chmod a+x /kotsadm-operator

EXPOSE 3000
# ARG version=unknown
# ENV VERSION=${version}
ENTRYPOINT ["/kotsadm-operator"]
