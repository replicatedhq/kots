FROM node:10 AS build
WORKDIR /src
ADD . /src

RUN make deps build
RUN rm -rf node_modules
RUN make deps-prod
RUN curl -L https://install.goreleaser.com/github.com/tj/node-prune.sh | bash && ./bin/node-prune

FROM node:10-buster-slim

# Install Kubectl
ENV KUBECTL_VERSION=v1.16.1
ENV KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_SHA256SUM=69cfb3eeaa0b77cc4923428855acdfc9ca9786544eeaff9c21913be830869d29
RUN curl -fsSLO "${KUBECTL_URL}" \
	&& echo "${KUBECTL_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_VERSION}" /usr/local/bin/kubectl

# Install kustomize
RUN curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/v2.0.3/kustomize_2.0.3_linux_amd64" > /tmp/kustomize && \
  chmod a+x /tmp/kustomize && \
  mv /tmp/kustomize /usr/local/bin

# Install kots.so
RUN curl -L "https://github.com/replicatedhq/kots/releases/download/v1.7.0-beta.1/kots.so_linux_amd64.tar.gz" > /tmp/kots.tar.gz && \
  cd /tmp && tar xzvf kots.tar.gz && \
  mv /tmp/kots.so /lib/kots.so && \
  rm -rf /tmp/*

# Install troubleshoot.so
RUN curl -L "https://github.com/replicatedhq/troubleshoot/releases/download/v0.9.14/troubleshoot.so_linux_amd64.tar.gz" > /tmp/troubleshoot.tar.gz && \
  cd /tmp && tar xzvf troubleshoot.tar.gz && \
  mv /tmp/troubleshoot.so /lib/troubleshoot.so && \
  rm -rf /tmp/*

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

ADD ./deploy/policy.json /etc/containers/policy.json
RUN apt-get -y update && apt-get install -y --no-install-recommends \
    libgpgme-dev libdevmapper-dev \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build /src/build /src/build
COPY --from=build /src/node_modules /src/node_modules

EXPOSE 3000
ARG commit=unknown
ENV COMMIT=${commit}

RUN useradd -c 'kotsadm-api user' -m -d /home/kotsadm-api -s /bin/bash -u 1001 kotsadm-api
RUN chown -R kotsadm-api.kotsadm-api /src
USER kotsadm-api
ENV HOME /home/kotsadm-api

CMD ["node", "/src/build/server/index.js"]
