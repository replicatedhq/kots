#!/bin/bash

set -e

# This script imports bucket content from the shared migration directory to the new minio instance.

# check if the migration has already been completed
if [ -f /export/.migration ];
then
    MIGRATION_DATE=$(cat /export/.migration)
    echo "migration already completed at $MIGRATION_DATE, no-op"
    exit 0
fi

# validate environment variables
if [ -z $KOTSADM_MINIO_MIGRATION_DIR ] ||
   [ -z $KOTSADM_MINIO_NEW_ALIAS ] ||
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

# create the new data directory
mkdir -p $KOTSADM_MINIO_MIGRATION_DIR/new-data

echo "starting minio instance"
/bin/sh -ce "/usr/bin/docker-entrypoint.sh minio -C /home/minio/.minio/ --quiet server $KOTSADM_MINIO_MIGRATION_DIR/new-data" &
MINIO_PID=$!

# wait for minio to be ready
until curl -s $KOTSADM_MINIO_ENDPOINT/minio/health/ready; do
    echo "waiting for minio to be ready"
    sleep 1
done

# alias the minio instance
echo "aliasing minio instance"
until $KOTSADM_MINIO_MIGRATION_DIR/bin/mc alias set $KOTSADM_MINIO_NEW_ALIAS $KOTSADM_MINIO_ENDPOINT $MINIO_ACCESS_KEY $MINIO_SECRET_KEY; do
    # minio may not actually be ready to accept requests immediately after it is "ready", so this provides a secondary check
    echo "attempting to alias minio instance"
    sleep 1
done

# check if the bucket already exists
if $KOTSADM_MINIO_MIGRATION_DIR/bin/mc ls $KOTSADM_MINIO_NEW_ALIAS | grep -q $KOTSADM_MINIO_BUCKET_NAME; then
    echo "bucket already exists, skipping creation"
else
    # create the bucket
    echo "creating minio bucket"
    $KOTSADM_MINIO_MIGRATION_DIR/bin/mc mb $KOTSADM_MINIO_NEW_ALIAS/$KOTSADM_MINIO_BUCKET_NAME
fi

# import the bucket content
if [ -d $KOTSADM_MINIO_MIGRATION_DIR/$KOTSADM_MINIO_BUCKET_NAME ]; then
    echo "importing minio bucket content"
    $KOTSADM_MINIO_MIGRATION_DIR/bin/mc mirror --preserve $KOTSADM_MINIO_MIGRATION_DIR/$KOTSADM_MINIO_BUCKET_NAME $KOTSADM_MINIO_NEW_ALIAS/$KOTSADM_MINIO_BUCKET_NAME
    echo "import complete"
else
    # if the directory does not exist, there is no bucket content to import
    echo "no bucket content to import"
fi

# shutdown minio
echo "stopping minio"
kill $MINIO_PID

# wait for minio to exit
wait $MINIO_PID

echo "minio stopped"

echo "replacing old data with new data"
shopt -s dotglob

if [ "$(ls -A /export)" ]; then
    echo "/export is not empty, moving old data to migration directory"
    mkdir -p $KOTSADM_MINIO_MIGRATION_DIR/old-data
    mv -v /export/* $KOTSADM_MINIO_MIGRATION_DIR/old-data/
else
    echo "/export is empty, no data to move to migration directory"
fi

mv -v $KOTSADM_MINIO_MIGRATION_DIR/new-data/* /export/

echo "adding migration complete marker"
date -u +"%Y-%m-%dT%H:%M:%SZ" > /export/.migration

echo "data migration complete"
