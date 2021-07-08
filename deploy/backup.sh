#!/bin/bash

set -e

# back up postgres data

export PG_BACKUP_FILE=/backup/kotsadm-postgres.sql
export PGPASSWORD=$POSTGRES_PASSWORD
pg_dump -U kotsadm -h kotsadm-postgres --clean > $PG_BACKUP_FILE

# back up s3 data if exists

if [ -n "$S3_ENDPOINT" ]; then
  export S3_DIR=/backup/s3/
  export S3_HOST=`echo $S3_ENDPOINT | awk -F/ '{print $3}'`
  rm -rf $S3_DIR
  mkdir -p $S3_DIR
  s3cmd --access_key=$S3_ACCESS_KEY_ID --secret_key=$S3_SECRET_ACCESS_KEY --host=$S3_HOST --no-ssl --host-bucket=$S3_BUCKET_NAME.$S3_HOST sync s3://$S3_BUCKET_NAME $S3_DIR
fi

# back up kotsadmdata volume if exists

export ARCHIVES_DIR=/kotsadmdata/archives
if [ -d $ARCHIVES_DIR ]; then
  # this is to work around the fact that Velero does not support backing up volumes of type hostpath (which could be the case in k3s clusters for example)
  # we copy the contents to the /backup directory where it gets restored later during the restore process to /kotsadmdata
  # ref: https://github.com/vmware-tanzu/velero/discussions/3378
  rm -rf /backup/archives
  cp -afrv $ARCHIVES_DIR /backup  
fi
