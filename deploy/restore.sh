#!bin/bash

set -e

# restore postgres data

export PG_BACKUP_FILE=/backup/kotsadm-postgres.sql
export PGPASSWORD=$POSTGRES_PASSWORD

if [ -f $PG_BACKUP_FILE ]; then
    psql -U kotsadm -h kotsadm-postgres -f $PG_BACKUP_FILE
    rm -f $PG_BACKUP_FILE
fi

# restore kotsadmdata volume

export ARCHIVES_BACKUP_DIR=/backup/archives

if [ -d $ARCHIVES_BACKUP_DIR ]; then
    cp -afrv $ARCHIVES_BACKUP_DIR /kotsadmdata
    rm -rf $ARCHIVES_BACKUP_DIR
fi
