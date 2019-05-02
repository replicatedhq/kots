#!/usr/bin/env node

import * as yargs from "yargs";

import * as migrateClusters from "./commands/migrate-clusters";
import * as ensureLocalCluster from "./commands/ensure-local-cluster";

yargs
  .env()
  .help()
  .command(
    migrateClusters.name,
    migrateClusters.describe,
    migrateClusters.builder,
    migrateClusters.handler,
  )
  .command(
    ensureLocalCluster.name,
    ensureLocalCluster.describe,
    ensureLocalCluster.builder,
    ensureLocalCluster.handler,
  )
  .argv;
