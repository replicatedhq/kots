#!bin/bash

set -e

export BACKUP_FILE=/backup/kotsadm-postgres.sql
export PGPASSWORD=$POSTGRES_PASSWORD
pg_dump -U kotsadm -h kotsadm-postgres --clean > $BACKUP_FILE
