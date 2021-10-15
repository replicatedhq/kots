FROM debian:buster-slim

## python-dateutil pkg is needed for s3cmd to work
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git gnupg2 python-dateutil \
  && for i in 1 2 3 4 5 6 7 8; do mkdir -p "/usr/share/man/man$i"; done \
  && curl --fail -N -s https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - \
  && echo "deb http://apt.postgresql.org/pub/repos/apt/ stretch-pgdg main" > /etc/apt/sources.list.d/PostgreSQL.list \
  && apt-get update && apt-get install -y --no-install-recommends postgresql-client-10 \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /usr/share/man/man*

# Install Kubectl 1.14
ENV KUBECTL_1_14_VERSION=v1.14.10
ENV KUBECTL_1_14_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_14_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_14_SHA256SUM=7729c6612bec76badc7926a79b26e0d9b06cc312af46dbb80ea7416d1fce0b36
RUN curl -fsSLO "${KUBECTL_1_14_URL}" \
	&& echo "${KUBECTL_1_14_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_14_VERSION}"

# Install Kubectl 1.16
ENV KUBECTL_1_16_VERSION=v1.16.15
ENV KUBECTL_1_16_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_16_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_16_SHA256SUM=e8913069293156ddf55f243814a22d2384fc18b165efb6200606fdeaad146605
RUN curl -fsSLO "${KUBECTL_1_16_URL}" \
	&& echo "${KUBECTL_1_16_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_16_VERSION}"

# Install Kubectl 1.17
ENV KUBECTL_1_17_VERSION=v1.17.17
ENV KUBECTL_1_17_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_17_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_17_SHA256SUM=8329fac94c66bf7a475b630972a8c0b036bab1f28a5584115e8dd26483de8349
RUN curl -fsSLO "${KUBECTL_1_17_URL}" \
	&& echo "${KUBECTL_1_17_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_17_VERSION}"

# Install Kubectl 1.18
ENV KUBECTL_1_18_VERSION=v1.18.20
ENV KUBECTL_1_18_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_18_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_18_SHA256SUM=66a9bb8e9843050340844ca6e72e67632b75b9ebb651559c49db22f35450ed2f
RUN curl -fsSLO "${KUBECTL_1_18_URL}" \
	&& echo "${KUBECTL_1_18_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_18_VERSION}"

# Install Kubectl 1.19
ENV KUBECTL_1_19_VERSION=v1.19.15
ENV KUBECTL_1_19_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_19_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_19_SHA256SUM=6f2ac7db8cfd59f660abc9891c1bb7da2dabd1cf5e114d836f2ffd39ee677d04
RUN curl -fsSLO "${KUBECTL_1_19_URL}" \
	&& echo "${KUBECTL_1_19_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_19_VERSION}"

# Install Kubectl 1.20
ENV KUBECTL_1_20_VERSION=v1.20.11
ENV KUBECTL_1_20_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_20_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_20_SHA256SUM=3a2bf981939df89f807858a481f6f5f2e33a7b9708bd029c8bece434db228efe
RUN curl -fsSLO "${KUBECTL_1_20_URL}" \
	&& echo "${KUBECTL_1_20_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_20_VERSION}"

# Install Kubectl 1.21
ENV KUBECTL_1_21_VERSION=v1.21.5
ENV KUBECTL_1_21_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_21_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_21_SHA256SUM=060ede75550c63bdc84e14fcc4c8ab3017f7ffc032fc4cac3bf20d274fab1be4
RUN curl -fsSLO "${KUBECTL_1_21_URL}" \
	&& echo "${KUBECTL_1_21_SHA256SUM} kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_21_VERSION}" \
	&& ln -s "/usr/local/bin/kubectl-${KUBECTL_1_21_VERSION}" /usr/local/bin/kubectl

# Install helm v3.4.2
RUN curl -L "https://get.helm.sh/helm-v3.4.2-linux-amd64.tar.gz" -o /tmp/helm.tar.gz && \
  tar -xzvf /tmp/helm.tar.gz && \
  rm /tmp/helm.tar.gz && \
  chmod a+x linux-amd64/helm && \
  mv linux-amd64/helm "/usr/local/bin/helm3.4.2"

# Install s3cmd
ENV S3CMD_VERSION=2.1.0
RUN curl -L https://github.com/s3tools/s3cmd/releases/download/v${S3CMD_VERSION}/s3cmd-${S3CMD_VERSION}.tar.gz > /tmp/s3cmd.tar.gz && \
  cd /tmp && tar -xzvf s3cmd.tar.gz && \
  rm s3cmd.tar.gz && \
  mv s3cmd-${S3CMD_VERSION}/s3cmd /usr/local/bin/s3cmd && \
  mv s3cmd-${S3CMD_VERSION}/S3 /usr/local/bin/S3 && \
  rm -rf s3cmd-${S3CMD_VERSION}

# Install kustomize 3
RUN curl -L "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz" > /tmp/kustomize.tar.gz && \
  tar -xzvf /tmp/kustomize.tar.gz && \
  rm /tmp/kustomize.tar.gz && \
  chmod a+x kustomize && \
  mv kustomize "/usr/local/bin/kustomize3.5.4"

# Setup user
RUN useradd -c 'kotsadm user' -m -d /home/kotsadm -s /bin/bash -u 1001 kotsadm
USER kotsadm
ENV HOME /home/kotsadm

COPY --chown=kotsadm:kotsadm ./deploy/backup.sh /backup.sh
COPY --chown=kotsadm:kotsadm ./deploy/restore-db.sh /restore-db.sh
COPY --chown=kotsadm:kotsadm ./deploy/restore-s3.sh /restore-s3.sh
COPY --chown=kotsadm:kotsadm ./deploy/restore.sh /restore.sh
COPY --chown=kotsadm:kotsadm ./deploy/migrate-s3.sh /migrate-s3.sh

COPY --chown=kotsadm:kotsadm ./deploy/fs-minio-check.sh /fs-minio-check.sh
COPY --chown=kotsadm:kotsadm ./deploy/fs-minio-reset.sh /fs-minio-reset.sh
COPY --chown=kotsadm:kotsadm ./deploy/fs-minio-keys-sha.sh /fs-minio-keys-sha.sh

COPY --chown=kotsadm:kotsadm ./deploy/s3-bucket-create.sh /s3-bucket-create.sh
COPY --chown=kotsadm:kotsadm ./deploy/s3-bucket-head.sh /s3-bucket-head.sh

COPY --chown=kotsadm:kotsadm ./bin/kotsadm /kotsadm

WORKDIR /

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}
ENTRYPOINT ["/kotsadm"]
CMD ["api"]
