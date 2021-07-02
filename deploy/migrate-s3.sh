#!/bin/bash

set -e
set -o pipefail

export ARCHIVES_DIR=/kotsadmdata/archives
export MIGRATION_FILE=$ARCHIVES_DIR/s3-migration.txt
if [ -f $MIGRATION_FILE ]; then
  echo 'migration has already run. no-op.'
  exit 0
fi

if [ -z $S3_ENDPOINT ] ||
   [ -z $S3_ACCESS_KEY_ID ] ||
   [ -z $S3_SECRET_ACCESS_KEY ] ||
   [ -z $S3_BUCKET_NAME ];
then
  echo 'no object store detected, skipping migration ...'
  exit 0
fi

export S3_HOST=`echo $S3_ENDPOINT | awk -F/ '{print $3}'`
export S3_S3CMD_FLAGS="--access_key=$S3_ACCESS_KEY_ID --secret_key=$S3_SECRET_ACCESS_KEY --host=$S3_HOST --no-ssl --host-bucket=$S3_BUCKET_NAME.$S3_HOST"
if s3cmd $S3_S3CMD_FLAGS ls s3://$S3_BUCKET_NAME 2>&1 | grep -q 'NoSuchBucket'; then
  echo 'bucket not found, skipping migration ...'
  exit 0
fi

echo 'object store and bucket detected, running migration ...'

mkdir -p $ARCHIVES_DIR
s3cmd $S3_S3CMD_FLAGS sync s3://$S3_BUCKET_NAME $ARCHIVES_DIR

echo 'migration ran successfully ...'
echo 'recording that the migration ran ...'
echo "migration ran at: $(date)" > "$MIGRATION_FILE"
