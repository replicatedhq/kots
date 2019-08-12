import pino from "pino";
import pinoPretty from "pino-pretty";

export const TSEDVerboseLogging = process.env.NODE_ENV !== "production" && process.env.NODE_ENV !== "staging" && !process.env.TSED_SUPPRESS_ACCESSLOG;

export const pinoLevel = process.env.LOG_LEVEL || "info";

function initLoggerFromEnv(): pino.Logger {
  const component = `kotsadm-api`;
  let options = {
    name: component,
    version: process.env.VERSION,
    level: pinoLevel,
    prettyPrint: {
      levelFirst: true,
      colorize: TSEDVerboseLogging,

    },
    prettifier: pinoPretty,
  };

  return pino(options);
}

export const logger = initLoggerFromEnv();
