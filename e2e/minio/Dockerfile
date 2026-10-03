# The MinIO Helm chart invokes mc from /usr/bin/mc in its post-install job and
# writes its config into a mounted emptyDir. Bitnami's legacy image keeps mc
# under /opt/bitnami and runs as an unprivileged user, so provide the expected
# path and run the short-lived setup job as root to write its mounted config.
FROM bitnamilegacy/minio-client:2025.7.21-debian-12-r3

USER 0
RUN test -x /opt/bitnami/minio-client/bin/mc && ln -s /opt/bitnami/minio-client/bin/mc /usr/bin/mc
