import * as fs from "fs";
import * as pino from "pino";
import * as pinoPretty from "pino-pretty";
import * as stream from "stream";

export const TSEDVerboseLogging = process.env.NODE_ENV !== "production" && process.env.NODE_ENV !== "staging" && !process.env.TSED_SUPPRESS_ACCESSLOG;

export const pinoLevel = process.env.PINO_LOG_LEVEL || process.env.LOG_LEVEL || "info";

function initLoggerFromEnv(): pino.Logger {
  const component = `ship-cluster-api`;
  let options = {
    name: component,
    version: process.env.VERSION,
    level: pinoLevel,
    prettifier: pinoPretty,
  };

  return pino(options);
}

export const logger = initLoggerFromEnv();
