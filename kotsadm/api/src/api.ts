#!/usr/bin/env node

import yargs from "yargs";

yargs
  .env()
  .help()
  .argv;
