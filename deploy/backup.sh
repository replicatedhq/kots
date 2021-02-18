#!bin/bash

set -e

export BACKUP_FILE=/backup/kotsadm-postgres.sql
export PGPASSWORD=$POSTGRES_PASSWORD
pg_dump -U kotsadm -h kotsadm-postgres --clean > $BACKUP_FILE

export S3_DIR=/backup/s3/
export S3_HOST=`echo $S3_ENDPOINT | awk -F/ '{print $3}'`
rm -rf $S3_DIR
mkdir -p $S3_DIR
s3cmd --access_key=$S3_ACCESS_KEY_ID --secret_key=$S3_SECRET_ACCESS_KEY --host=$S3_HOST --no-ssl --host-bucket=$S3_BUCKET_NAME.$S3_HOST sync s3://$S3_BUCKET_NAME $S3_DIR
