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

export KOTSADM_DATA_BACKUP_DIR=/backup/kotsadmdata

if [ -d $KOTSADM_DATA_BACKUP_DIR ]; then
    cp -R $KOTSADM_DATA_BACKUP_DIR/. /kotsadmdata
    rm -rf $KOTSADM_DATA_BACKUP_DIR
fi
