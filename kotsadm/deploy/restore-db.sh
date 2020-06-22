#!bin/bash

set -e

export BACKUP_FILE=/backup/kotsadm-postgres.sql
export PGPASSWORD=$POSTGRES_PASSWORD

if [ ! -f $BACKUP_FILE ]; then
    exit 0
fi

psql -U kotsadm -h kotsadm-postgres -f $BACKUP_FILE
rm -f $BACKUP_FILE