#!bin/bash

set -e

export S3_DIR=/backup/s3/
export S3_HOST=`echo $S3_ENDPOINT | awk -F/ '{print $3}'`

if [ ! -d $S3_DIR ]; then
    exit 0
fi

export S3CMD_FLAGS="--access_key=$S3_ACCESS_KEY_ID --secret_key=$S3_SECRET_ACCESS_KEY --host=$S3_HOST --no-ssl --host-bucket=$S3_BUCKET_NAME.$S3_HOST"

if s3cmd $S3CMD_FLAGS ls s3://$S3_BUCKET_NAME 2>&1 | grep -q 'NoSuchBucket'
then
    echo "bucket $S3_BUCKET_NAME does not exist, creating ..."
    s3cmd $S3CMD_FLAGS mb s3://$S3_BUCKET_NAME
fi

s3cmd $S3CMD_FLAGS sync $S3_DIR s3://$S3_BUCKET_NAME
rm -rf $S3_DIR
