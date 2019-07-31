#!/usr/bin/env node

import yargs from "yargs";

import * as ensureLocalCluster from "./commands/ensure-local-cluster";
import * as migrateDownstreamClusterUsers from "./commands/migrate-downstream-cluster-users";
import * as syncPrStatus from "./commands/sync-pr-status";
yargs
  .env()
  .help()
  .command(
    ensureLocalCluster.name,
    ensureLocalCluster.describe,
    ensureLocalCluster.builder,
    ensureLocalCluster.handler,
  )
  .command(
    migrateDownstreamClusterUsers.name,
    migrateDownstreamClusterUsers.describe,
    migrateDownstreamClusterUsers.builder,
    migrateDownstreamClusterUsers.handler
  )
  .command(
    syncPrStatus.name,
    syncPrStatus.describe,
    syncPrStatus.builder,
    syncPrStatus.handler
  )
  .option("dryRun", {
    alias: "d",
    description: "See output only, don't overide any data",
    type: "boolean",
  })
  .argv;
