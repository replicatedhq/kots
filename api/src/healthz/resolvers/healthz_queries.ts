import { logger } from "../../server/logger";

export function HealthzQueries(stores: any) {
  return {
    async healthz(): Promise<{}> {
      return {
        version: process.env.VERSION || "unknown",
      };
    },

    async ping(): Promise<string> {
      logger.info("got ping");

      return "pong";
    }
  }
}
