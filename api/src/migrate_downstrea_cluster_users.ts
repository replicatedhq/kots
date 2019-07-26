import yargs from "yargs";

import * as migrateDownstreamClusterUsers from "./commands/migrate-downstream-cluster-users";

yargs
  .env()
  .help()
  .command(
    migrateDownstreamClusterUsers.name,
    migrateDownstreamClusterUsers.describe,
    migrateDownstreamClusterUsers.builder,
    migrateDownstreamClusterUsers.handler
  )
  .argv;
