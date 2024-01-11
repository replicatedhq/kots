#!/bin/bash

set -e

# This script copies the minio client binary to a dedicated migration directory
# This is necessary because the minio client binary is not available in older versions of the minio image

# check if the migration has already been completed
if [ -f /export/.migration-complete ];
then
    MIGRATION_DATE=$(cat /export/.migration-complete)
    echo "migration already completed at $MIGRATION_DATE, no-op"
    exit 0
fi

# validate environment variables
if [ -z $KOTSADM_MINIO_MIGRATION_DIR ]; then
    echo 'KOTSADM_MINIO_MIGRATION_DIR not set'
    exit 1
fi

echo "copying minio client binary to migration directory and preserving permissions"
mkdir -p $KOTSADM_MINIO_MIGRATION_DIR/bin
cp /usr/bin/mc $KOTSADM_MINIO_MIGRATION_DIR/bin/mc
