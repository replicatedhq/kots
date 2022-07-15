#!/bin/bash

set -e

# This script triggers the upgrade process for upgrading postgres 10 to 14
# - no-op if the data directory for the postgres 10 instance does not exist (new installs & post upgrade).
# - pg_upgrade requires both data directories to have 700 permissions.
# - remove postgres 14 data directory if exists because it's automatically created and configured by the entrypoint script, and will fail otherwise.
# - run the docker entrypoint script to initialize and configure the postgres 14 instance.
# - pg_upgrade will fail if the database already exists in the postgres 14 instance, and the docker entrypoint script creates it automatically, so:
#     * start the postgres 14 server.
#     * wait for the server to be ready.
#     * connect to the default 'template1' db provided by postgres (because postgres won't allow you to drop a db if you're connected to it).
#     * drop the database that was created by the docker entrypoint script.
# - pg_upgrade requires both servers to be stopped, so stop the postgres 14 instance that was started.
# - in some cases, the postgres 10 instance may not have shut down cleanly, which causes pg_upgrade to fail. start it and shut it down properly.
# - pg_upgrade has to be run inside a directory where the user has write permissions, so cd into the dedicated 'upgrade' directory.
# - run the pg_upgrade command.
# - the pg_upgrade command will generate a 'delete_old_cluster.sh' script that can be used to delete the data of the old postgres instance. run it.
# - remove the postgres 10 instance's server binaries directory.

if [ ! -d $PGDATAOLD ]; then
  echo 'postgres 10 data not detected. no-op.'
  exit 0
fi

chmod 700 $PGDATAOLD
rm -rf $PGDATA

docker-entrypoint.sh postgres &

while ! pg_isready -U $POSTGRES_USER -h 127.0.0.1 -p $PGPORT; do sleep 1; done
psql -U $POSTGRES_USER -c '\connect template1' -c "drop database $POSTGRES_DB with (force)"
pg_ctl stop -w

$PGBINOLD/pg_ctl start -w -D $PGDATAOLD
$PGBINOLD/pg_ctl stop -w -D $PGDATAOLD

cd $POSTGRES_UPGRADE_DIR
pg_upgrade -U $POSTGRES_USER -v

./delete_old_cluster.sh
rm -rf pg10
