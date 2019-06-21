import { logger } from "../../server/logger";
import { Stores } from "../../schema/stores";

export function HealthzQueries(stores: Stores) {
  return {
    async healthz(): Promise<{}> {
      return {
        version: process.env.VERSION || "unknown",
      };
    },

    async ping(): Promise<string> {
      logger.info({msg: "got ping"});

      return "pong";
    }
  }
}
