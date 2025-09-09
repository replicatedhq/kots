#!/bin/bash

set -e

# restore rqlite data

export RQLITE_BACKUP_FILE=/backup/kotsadm-rqlite.sql
export RQLITE_USERNAME=kotsadm
export RQLITE_HOSTNAME=kotsadm-rqlite
export RQLITE_PORT=4001

if [ -f $RQLITE_BACKUP_FILE ]; then
    curl -v -f http://"$RQLITE_USERNAME":"$RQLITE_PASSWORD"@"$RQLITE_HOSTNAME":"$RQLITE_PORT"/db/load -H "Content-type: application/octet-stream" --data-binary @$RQLITE_BACKUP_FILE
    rm -f $RQLITE_BACKUP_FILE
    echo ""
fi

# restore kotsadmdata volume

export ARCHIVES_BACKUP_DIR=/backup/archives

if [ -d $ARCHIVES_BACKUP_DIR ]; then
    cp -afrv $ARCHIVES_BACKUP_DIR /kotsadmdata
    rm -rf $ARCHIVES_BACKUP_DIR
fi
