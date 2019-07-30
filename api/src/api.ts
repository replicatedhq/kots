#!/usr/bin/env node

import yargs from "yargs";

import * as ensureLocalCluster from "./commands/ensure-local-cluster";
import * as migrateDownstreamClusterUsers from "./commands/migrate-downstream-cluster-users";
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
  .argv;
