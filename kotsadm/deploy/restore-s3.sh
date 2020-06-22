#!bin/bash

set -e

export S3_DIR=/backup/s3/
export S3_HOST=`echo $S3_ENDPOINT | awk -F/ '{print $3}'`

if [ ! -f $S3_DIR ]; then
    exit 0
fi

s3cmd --access_key=$S3_ACCESS_KEY_ID --secret_key=$S3_SECRET_ACCESS_KEY --host=$S3_HOST --no-ssl --host-bucket=$S3_BUCKET_NAME.$S3_HOST sync $S3_DIR s3://$S3_BUCKET_NAME
rm -rf $S3_DIR
