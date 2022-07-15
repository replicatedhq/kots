FROM debian:buster-slim

## python-dateutil pkg is needed for s3cmd to work
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git gnupg2 python-dateutil \
  && for i in 1 2 3 4 5 6 7 8; do mkdir -p "/usr/share/man/man$i"; done \
  && curl --fail -N -s https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - \
  && echo "deb http://apt.postgresql.org/pub/repos/apt/ buster-pgdg main" > /etc/apt/sources.list.d/PostgreSQL.list \
  && apt-get update && apt-get install -y --no-install-recommends postgresql-client-10 \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /usr/share/man/man*

# KOTS can be configured to use a specific version of kubectl by setting kubectlVersion in the
# kots.io/v1beta1.Application spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all kubectl binaries in the KOTS_KUBECTL_BIN_DIR directory for use by KOTS.

ENV KOTS_KUBECTL_BIN_DIR=/usr/local/bin

# The following kubectl versions are automatically generated. Run `make generate-kubectl-versions` to update.

# __BEGIN_KUBECTL_VERSIONS__

# Install Kubectl 1.22
ENV KUBECTL_1_22_VERSION=v1.22.12
ENV KUBECTL_1_22_URL=https://dl.k8s.io/release/${KUBECTL_1_22_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_22_SHA256SUM=8e36c8fa431e454e3368c6174ce3111b7f49c28feebdae6801ab3ca45f02d352
RUN curl -fsSLO "${KUBECTL_1_22_URL}" \
  && echo "${KUBECTL_1_22_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.22"

# Install Kubectl 1.23
ENV KUBECTL_1_23_VERSION=v1.23.9
ENV KUBECTL_1_23_URL=https://dl.k8s.io/release/${KUBECTL_1_23_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_23_SHA256SUM=053561f7c68c5a037a69c52234e3cf1f91798854527692acd67091d594b616ce
RUN curl -fsSLO "${KUBECTL_1_23_URL}" \
  && echo "${KUBECTL_1_23_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.23"

# Install Kubectl 1.24
ENV KUBECTL_1_24_VERSION=v1.24.3
ENV KUBECTL_1_24_URL=https://dl.k8s.io/release/${KUBECTL_1_24_VERSION}/bin/linux/amd64/kubectl
ENV KUBECTL_1_24_SHA256SUM=8a45348bdaf81d46caf1706c8bf95b3f431150554f47d444ffde89e8cdd712c1
RUN curl -fsSLO "${KUBECTL_1_24_URL}" \
  && echo "${KUBECTL_1_24_SHA256SUM} kubectl" | sha256sum -c - \
  && chmod +x kubectl \
  && mv kubectl "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.24" \
  && ln -s "${KOTS_KUBECTL_BIN_DIR}/kubectl-v1.24" "${KOTS_KUBECTL_BIN_DIR}/kubectl"

# __END_KUBECTL_VERSIONS__

# KOTS can be configured to use a specific version of kustomize by setting kustomizeVersion in the
# kots.io/v1beta1.Application spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all kustomize binaries in the KOTS_KUSTOMIZE_BIN_DIR directory for use by KOTS.
# CURRENTLY ONLY ONE VERSION IS SHIPPED BELOW

ENV KOTS_KUSTOMIZE_BIN_DIR=/usr/local/bin

# Install kustomize 3
ENV KUSTOMIZE3_VERSION=3.5.4
ENV KUSTOMIZE3_URL=https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE3_VERSION}/kustomize_v${KUSTOMIZE3_VERSION}_linux_amd64.tar.gz
ENV KUSTOMIZE3_SHA256SUM=5cdeb2af81090ad428e3a94b39779b3e477e2bc946be1fe28714d1ca28502f6a
RUN curl -fsSL -o kustomize.tar.gz "${KUSTOMIZE3_URL}" \
  && echo "${KUSTOMIZE3_SHA256SUM} kustomize.tar.gz" | sha256sum -c - \
  && tar -xzvf kustomize.tar.gz \
  && rm kustomize.tar.gz \
  && chmod a+x kustomize \
  && mv kustomize "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize${KUSTOMIZE3_VERSION}" \
  && ln -s "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize${KUSTOMIZE3_VERSION}" "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize3" \
  && ln -s "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize3" "${KOTS_KUSTOMIZE_BIN_DIR}/kustomize"

# KOTS can be configured to use a specific version of helm by setting helmVersion in the
# kots.io/v1beta1.HelmChart spec. The github.com/replicatedhq/kots/pkg/binaries package will
# discover all helm binaries in the KOTS_HELM_BIN_DIR directory for use by KOTS.

ENV KOTS_HELM_BIN_DIR=/usr/local/bin

# Install helm v3
ENV HELM3_VERSION=3.8.2
ENV HELM3_URL=https://get.helm.sh/helm-v${HELM3_VERSION}-linux-amd64.tar.gz
ENV HELM3_SHA256SUM=6cb9a48f72ab9ddfecab88d264c2f6508ab3cd42d9c09666be16a7bf006bed7b
RUN cd /tmp && curl -fsSL -o helm.tar.gz "${HELM3_URL}" \
  && echo "${HELM3_SHA256SUM} helm.tar.gz" | sha256sum -c - \
  && tar -xzvf helm.tar.gz \
  && chmod a+x linux-amd64/helm \
  && mv linux-amd64/helm "${KOTS_HELM_BIN_DIR}/helm${HELM3_VERSION}" \
  && ln -s "${KOTS_HELM_BIN_DIR}/helm${HELM3_VERSION}" "${KOTS_HELM_BIN_DIR}/helm3" \
  && ln -s "${KOTS_HELM_BIN_DIR}/helm3" "${KOTS_HELM_BIN_DIR}/helm" \
  && rm -rf helm.tar.gz linux-amd64

# Install s3cmd
ENV S3CMD_VERSION=2.1.0
ENV S3CMD_URL=https://github.com/s3tools/s3cmd/releases/download/v${S3CMD_VERSION}/s3cmd-${S3CMD_VERSION}.tar.gz
RUN cd /tmp && curl -fsSL -o s3cmd.tar.gz "${S3CMD_URL}" \
  && curl -fsSL -o s3cmd.tar.gz.asc "${S3CMD_URL}.asc" \
  && gpg --keyserver keyserver.ubuntu.com --recv-keys 0x0d37a8f4a5d183d5541d85d9241769189ac3d00b \
  && cat s3cmd.tar.gz.asc \
  && gpg --verify s3cmd.tar.gz.asc s3cmd.tar.gz \
  && tar -xzvf s3cmd.tar.gz \
  && mv s3cmd-${S3CMD_VERSION}/s3cmd /usr/local/bin/s3cmd \
  && mv s3cmd-${S3CMD_VERSION}/S3 /usr/local/bin/S3 \
  && rm -rf s3cmd.tar.gz s3cmd.tar.gz.asc s3cmd-${S3CMD_VERSION}

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

COPY --chown=kotsadm:kotsadm ./deploy/kots-upgrade.sh /kots-upgrade.sh

COPY --chown=kotsadm:kotsadm ./bin/kotsadm /kotsadm
COPY --chown=kotsadm:kotsadm ./bin/kots /kots

WORKDIR /

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}
ENTRYPOINT ["/kotsadm"]
CMD ["api"]
