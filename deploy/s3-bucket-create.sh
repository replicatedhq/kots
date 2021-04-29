#!/bin/bash

set -e

TMP_S3_S3CMD_SSL_FLAGS=""
if [ -z "$TMP_CA_CERT" ] && echo "$TMP_S3_ENDPOINT" | grep -q "http://"; then
  TMP_S3_S3CMD_SSL_FLAGS="--no-ssl"
else
    echo "$TMP_CA_CERT" > /tmp/cabundle.crt
    TMP_S3_S3CMD_SSL_FLAGS="--ca-certs=/tmp/cabundle.crt"
fi

TMP_S3_HOST=$(echo "$TMP_S3_ENDPOINT" | awk -F/ '{print $3}')
export TMP_S3_HOST
export TMP_S3_S3CMD_FLAGS="--access_key=$TMP_S3_ACCESS_KEY_ID --secret_key=$TMP_S3_SECRET_ACCESS_KEY --host=$TMP_S3_HOST --host-bucket=$TMP_S3_BUCKET_NAME.$TMP_S3_HOST $TMP_S3_S3CMD_SSL_FLAGS"

echo "Running with flags: --access_key=***** --secret_key=***** --host=$TMP_S3_HOST --host-bucket=$TMP_S3_BUCKET_NAME.$TMP_S3_HOST $TMP_S3_S3CMD_SSL_FLAGS"

if s3cmd $TMP_S3_S3CMD_FLAGS ls s3://$TMP_S3_BUCKET_NAME 2>&1 | grep -q 'NoSuchBucket'
then
    s3cmd $TMP_S3_S3CMD_FLAGS mb s3://$TMP_S3_BUCKET_NAME
fi
echo '{"success": true}'
