FROM minio/minio:RELEASE.2019-10-12T01-39-57Z

RUN apk update && apk add ca-certificates libcap && rm -rf /var/cache/apk/*

# /bin/busybox is the backend to /bin/chown, and so this allows any user to change any file ownership
# this in turn allows the init container to take ownership of the data directory, so that minio can run
RUN setcap cap_chown=+ep /bin/busybox

RUN adduser -h /home/minio -s /bin/sh -u 1001 -D minio
USER minio
ENV HOME /home/minio
