#!/bin/bash

set -e

# This script imports bucket content from the shared migration directory to the new minio instance.

# check if the migration has already been completed
if [ -f /export/.migration-complete ];
then
    MIGRATION_DATE=$(cat /export/.migration-complete)
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

# now that we have a copy of the old minio data in the migration directory, remove the old data from the volume
echo "removing old minio data"
shopt -s dotglob
rm -rfv /export/*

echo "starting new minio instance"
minio -C /home/minio/.minio/ server /export &
MINIO_PID=$!

# alias the minio instance
echo "aliasing minio instance"
until $KOTSADM_MINIO_MIGRATION_DIR/bin/mc alias set $KOTSADM_MINIO_NEW_ALIAS $KOTSADM_MINIO_ENDPOINT $MINIO_ACCESS_KEY $MINIO_SECRET_KEY; do
    # minio may not actually be ready to accept requests immediately after it is "ready", so this provides a secondary check
    echo "attempting to alias minio instance"
    sleep 1
done

# wait for minio to be ready
until $KOTSADM_MINIO_MIGRATION_DIR/bin/mc ready $KOTSADM_MINIO_NEW_ALIAS; do
    echo "waiting for minio to be ready"
    sleep 1
done

# check if the bucket already exists
if $KOTSADM_MINIO_MIGRATION_DIR/bin/mc ls $KOTSADM_MINIO_NEW_ALIAS/$KOTSADM_MINIO_BUCKET_NAME; then
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

# mark the migration as complete
echo "marking migration as complete"
date -u +"%Y-%m-%dT%H:%M:%SZ" > /export/.migration-complete

# clean the migration directory
echo "cleaning up migration directory"
rm -rf $KOTSADM_MINIO_MIGRATION_DIR/*

# shutdown minio
echo "stopping minio"
kill $MINIO_PID

# wait for minio to exit
wait $MINIO_PID

echo "minio stopped"
