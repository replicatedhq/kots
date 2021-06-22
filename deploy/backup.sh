#!bin/bash

set -e

export PG_BACKUP_FILE=/backup/kotsadm-postgres.sql
export PGPASSWORD=$POSTGRES_PASSWORD
pg_dump -U kotsadm -h kotsadm-postgres --clean > $PG_BACKUP_FILE

# this is to work around the fact that Velero does not support backing up volumes of type hostpath (which could be the case in k3s clusters for example)
# we copy the contents to the /backup directory where it gets restored later during the restore process to /kotsadmdata
# ref: https://github.com/vmware-tanzu/velero/discussions/3378
rm -rf /backup/kotsadmdata
cp -afRv /kotsadmdata /backup
