import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function MonitoringMutations(stores: Stores) {
  return {
    async setPrometheusAddress(root: any, args: any, context: Context): Promise<boolean> {
      const { value } = args;
      await stores.paramsStore.setParam("PROMETHEUS_ADDRESS", value);
      return true;
    },

    async deletePrometheusAddress(root: any, args: any, context: Context): Promise<boolean> {
      await stores.paramsStore.deleteParam("PROMETHEUS_ADDRESS");
      return true;
    },
  }
}
