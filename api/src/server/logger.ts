import pino from "pino";
import pinoPretty from "pino-pretty";

export const TSEDVerboseLogging = process.env.NODE_ENV !== "production" && process.env.NODE_ENV !== "staging" && !process.env.TSED_SUPPRESS_ACCESSLOG;

export const pinoLevel = process.env.LOG_LEVEL || "info";

function initLoggerFromEnv(): pino.Logger {
  return pino({
    name: "kotsadm-api",
    timestamp: () => {
      console.log(arguments);
      return (new Date()).toISOString();
    },
    level: pinoLevel,
    prettyPrint: {
      levelFirst: true,
      forceColor: TSEDVerboseLogging,
    },
  });
}

export const logger = initLoggerFromEnv();
