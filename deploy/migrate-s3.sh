#!bin/bash

set -e

export MIGRATION_FILE=/kotsadmdata/s3-migration.txt
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

echo 'object store detected, running migration ...'

export DEST_DIR=/kotsadmdata/
export S3_HOST=`echo $S3_ENDPOINT | awk -F/ '{print $3}'`
s3cmd --access_key=$S3_ACCESS_KEY_ID --secret_key=$S3_SECRET_ACCESS_KEY --host=$S3_HOST --no-ssl --host-bucket=$S3_BUCKET_NAME.$S3_HOST sync s3://$S3_BUCKET_NAME $DEST_DIR

echo 'migration ran successfully ...'
echo 'recording that the migration ran ...'
echo "migration ran at: $(date)" > "$MIGRATION_FILE"
