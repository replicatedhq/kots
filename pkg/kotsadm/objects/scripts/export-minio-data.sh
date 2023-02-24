#!/bin/bash

set -e

# This script exports bucket content from the old minio instance to the shared migration directory

# check if the migration has already been completed
if [ -f /export/.migration ];
then
    MIGRATION_DATE=$(cat /export/.migration)
    echo "migration already completed at $MIGRATION_DATE, no-op"
    exit 0
elif [ -f /export/.export-complete ];
then
    EXPORT_DATE=$(cat /export/.export-complete)
    echo "export already completed at $EXPORT_DATE, no-op"
    exit 0
fi

# validate environment variables
if [ -z $KOTSADM_MINIO_MIGRATION_DIR ] ||
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
cd $KOTSADM_MINIO_MIGRATION_DIR

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
until $KOTSADM_MINIO_MIGRATION_DIR/bin/mc alias set $KOTSADM_MINIO_LEGACY_ALIAS $KOTSADM_MINIO_ENDPOINT $MINIO_ACCESS_KEY $MINIO_SECRET_KEY; do
    # minio may not actually be ready to accept requests immediately after it is "ready", so this provides a secondary check
    echo "attempting to alias minio instance"
    sleep 1
done

# check if the bucket exists and export the content
if $KOTSADM_MINIO_MIGRATION_DIR/bin/mc ls $KOTSADM_MINIO_LEGACY_ALIAS | grep -q $KOTSADM_MINIO_BUCKET_NAME; then
    echo "exporting minio bucket content"
    $KOTSADM_MINIO_MIGRATION_DIR/bin/mc mirror --preserve $KOTSADM_MINIO_LEGACY_ALIAS/$KOTSADM_MINIO_BUCKET_NAME $KOTSADM_MINIO_MIGRATION_DIR/$KOTSADM_MINIO_BUCKET_NAME

    echo "export complete"
else
    echo "bucket does not exist, skipping export"
fi

# shutdown minio
echo "stopping minio"
kill $MINIO_PID

# wait for minio to exit
wait $MINIO_PID

echo "minio stopped"

# mark the export as complete
echo "marking export as complete"
date -u +"%Y-%m-%dT%H:%M:%SZ" > /export/.export-complete
