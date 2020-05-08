import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function MonitoringQueries(stores: Stores) {
  return {
    async getPrometheusAddress(root: any, args: any, context: Context): Promise<string | null> {
      return await stores.paramsStore.getParam("PROMETHEUS_ADDRESS");
    },
  };
}
