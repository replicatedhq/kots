FROM debian:buster-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git gnupg2 s3cmd \
  && for i in 1 2 3 4 5 6 7 8; do mkdir -p "/usr/share/man/man$i"; done \
  && curl --fail -N -s https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - \
  && echo "deb http://apt.postgresql.org/pub/repos/apt/ stretch-pgdg main" > /etc/apt/sources.list.d/PostgreSQL.list \
  && apt-get update && apt-get install -y --no-install-recommends postgresql-client-10 \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /usr/share/man/man*

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

COPY --chown=kotsadm:kotsadm ./deploy/fs-minio-check.sh /fs-minio-check.sh
COPY --chown=kotsadm:kotsadm ./deploy/fs-minio-reset.sh /fs-minio-reset.sh
COPY --chown=kotsadm:kotsadm ./deploy/fs-minio-keys-sha.sh /fs-minio-keys-sha.sh

COPY --chown=kotsadm:kotsadm ./deploy/s3-bucket-create.sh /s3-bucket-create.sh
COPY --chown=kotsadm:kotsadm ./deploy/s3-bucket-head.sh /s3-bucket-head.sh

COPY --chown=kotsadm:kotsadm ./bin/kotsadm /kotsadm
COPY --chown=kotsadm:kotsadm ./web/dist /web/dist

WORKDIR /

EXPOSE 3000
ARG version=unknown
ENV VERSION=${version}
ENTRYPOINT ["/kotsadm"]
CMD ["api"]
