#!/bin/bash

set -e

export BACKUP_FILE=/backup/kotsadm-rqlite.sql
export RQLITE_USERNAME=kotsadm
export RQLITE_HOSTNAME=kotsadm-rqlite
export RQLITE_PORT=4001

if [ ! -f $BACKUP_FILE ]; then
    exit 0
fi

curl -v -f http://"$RQLITE_USERNAME":"$RQLITE_PASSWORD"@"$RQLITE_HOSTNAME":"$RQLITE_PORT"/db/load -H "Content-type: application/octet-stream" --data-binary @$BACKUP_FILE
rm -f $BACKUP_FILE
