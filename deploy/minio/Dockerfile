FROM minio/minio:RELEASE.2019-10-12T01-39-57Z

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

RUN adduser -h /home/minio -s /bin/sh -u 1001 -D minio
USER minio
ENV HOME /home/minio
