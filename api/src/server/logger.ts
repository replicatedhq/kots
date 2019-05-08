import * as fs from "fs";
import * as pino from "pino";
import * as pinoPretty from "pino-pretty";
import * as stream from "stream";

export const TSEDVerboseLogging = process.env.NODE_ENV !== "production" && process.env.NODE_ENV !== "staging" && !process.env.TSED_SUPPRESS_ACCESSLOG;

export const pinoLevel = process.env.PINO_LOG_LEVEL || process.env.LOG_LEVEL || "info";

function initLoggerFromEnv(): pino.Logger {
  const dest = process.env.LOG_FILE ? fs.createWriteStream(process.env.LOG_FILE) : process.stdout;

  const component = `ship-cluster-api`;
  let options = {
    name: component,
    level: pinoLevel,
    prettifier: undefined,
  };

  if (!process.env.PINO_LOG_PRETTY) {
    return pino(options, dest as stream.Writable).child({
      version: process.env.VERSION,
      component,
    });
  }

  options.prettifier = pinoPretty;
  return pino(options);
}

export const logger = initLoggerFromEnv();
