#!/bin/bash

set -e

# This script copies the postgres 10 server binary files that are needed for the upgrade.
# pg_upgrade requires the server binaries for the version of postgres to be upgraded, so copy it to the dedicated 'upgrade' directory.

if [ ! -d $PGDATA ]; then
  echo 'postgres 10 data not detected. no-op.'
  exit 0
fi

mkdir -p $POSTGRES_UPGRADE_DIR/pg10
cp -frv /usr/local/* $POSTGRES_UPGRADE_DIR/pg10
