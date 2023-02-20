#!/bin/bash

set -e

# This script exports configuration, bucket metadata, iam settings, and bucket content from the old minio instance
# to the shared migration directory

# check if the migration has already been completed
if [ -f /export/.migration ];
then
    echo "migration already completed, no-op"
    exit 0
fi

# validate environment variables
if [ -z $KOTSADM_MINIO_MIGRATION_WORK_DIR ] ||
   [ -z $KOTSADM_MINIO_LEGACY_ALIAS ] ||
   [ -z $KOTSADM_MINIO_ENDPOINT ] ||
   [ -z $MINIO_ACCESS_KEY ] ||
   [ -z $MINIO_SECRET_KEY ] ||
   [ -z $KOTSADM_MINIO_BUCKET_NAME ];
then
    echo 's3 migration environment variables not set'
    exit 1
fi

# change the working directory to the migration directory
cd $KOTSADM_MINIO_MIGRATION_WORK_DIR

echo "starting minio instance"
/bin/sh -ce "/usr/bin/docker-entrypoint.sh minio -C /home/minio/.minio/ server /export" &
MINIO_PID=$!

# wait for minio to be ready
until curl -s $KOTSADM_MINIO_ENDPOINT/minio/health/ready; do
    echo "waiting for minio to be ready"
    sleep 1
done

# alias the minio instance
echo "aliasing minio instance"
until $KOTSADM_MINIO_MIGRATION_WORK_DIR/bin/mc alias set $KOTSADM_MINIO_LEGACY_ALIAS $KOTSADM_MINIO_ENDPOINT $MINIO_ACCESS_KEY $MINIO_SECRET_KEY; do
    # minio may not actually be ready to accept requests immediately after it is "ready", so this provides a secondary check
    echo "attempting to alias minio instance"
    sleep 1
done

echo "exporting minio bucket content"
$KOTSADM_MINIO_MIGRATION_WORK_DIR/bin/mc mirror --preserve $KOTSADM_MINIO_LEGACY_ALIAS/$KOTSADM_MINIO_BUCKET_NAME $KOTSADM_MINIO_MIGRATION_WORK_DIR/$KOTSADM_MINIO_BUCKET_NAME

echo "export complete"

# shutdown minio
echo "stopping minio"
kill $MINIO_PID

# wait for minio to exit
wait $MINIO_PID

echo "minio stopped"
