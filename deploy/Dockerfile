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
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_19_VERSION}"

ENV KUBECTL_1_20_VERSION=v1.20.4
ENV KUBECTL_1_20_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_20_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_20_SHA256SUM=98e8aea149b00f653beeb53d4bd27edda9e73b48fed156c4a0aa1dabe4b1794c
RUN curl -fsSLO "${KUBECTL_1_20_URL}" \
	&& echo "${KUBECTL_1_20_SHA256SUM}  kubectl" | sha256sum -c - \
	&& chmod +x kubectl \
	&& mv kubectl "/usr/local/bin/kubectl-${KUBECTL_1_20_VERSION}"

ENV KUBECTL_1_21_VERSION=v1.21.2
ENV KUBECTL_1_21_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_1_21_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_21_SHA256SUM=55b982527d76934c2f119e70bf0d69831d3af4985f72bb87cd4924b1c7d528da
RUN curl -fsSLO "${KUBECTL_1_21_URL}" \
	&& echo "${KUBECTL_1_21_SHA256SUM}  kubectl" | sha256sum -c - \
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
