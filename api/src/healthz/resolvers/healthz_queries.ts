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
      logger.debug({msg: "got ping"});

      return "pong";
    }
  }
}
