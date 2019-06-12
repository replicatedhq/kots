#!/usr/bin/env node

import * as yargs from "yargs";

import * as ensureLocalCluster from "./commands/ensure-local-cluster";

yargs
  .env()
  .help()
  .command(
    ensureLocalCluster.name,
    ensureLocalCluster.describe,
    ensureLocalCluster.builder,
    ensureLocalCluster.handler,
  )
  .argv;
