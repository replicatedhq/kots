#!bin/bash

set -e

export TMP_S3_HOST=`echo $TMP_S3_ENDPOINT | awk -F/ '{print $3}'`
export TMP_S3_S3CMD_FLAGS="--access_key=$TMP_S3_ACCESS_KEY_ID --secret_key=$TMP_S3_SECRET_ACCESS_KEY --host=$TMP_S3_HOST --no-ssl --host-bucket=$TMP_S3_BUCKET_NAME.$TMP_S3_HOST"

if s3cmd $TMP_S3_S3CMD_FLAGS ls s3://$TMP_S3_BUCKET_NAME 2>&1 | grep -q 'NoSuchBucket'
then
    s3cmd $TMP_S3_S3CMD_FLAGS mb s3://$TMP_S3_BUCKET_NAME
fi

echo '{"success": true}'
