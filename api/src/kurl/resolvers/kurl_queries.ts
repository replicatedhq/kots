import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";
import * as k8s from "@kubernetes/client-node";
import _ from "lodash";
import { logger } from "../../server/logger";

export function KurlQueries(stores: Stores, params: Params) {
  return {
    async kurl(root: any, args: any, context: Context): Promise<any> {
      context.requireSingleTenantSession();

      if (!params.enableKurl) {
        return {
          addNodeCommand: "",
          nodes: [],
        };
      }

      // this isn't stored in the database, it's read in realtime
      // from the cluster

      try {
        const kc = new k8s.KubeConfig();
        kc.loadFromDefault();
        const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

        const res = await k8sApi.listNode();
        const nodes = _.map(res.body.items, (item) => {
          const memoryPressureCondition = _.find(item.status!.conditions!, { type: "MemoryPressure" });
          const diskPressureCondition = _.find(item.status!.conditions!, { type: "DiskPressure" });
          const pidPressureCondition = _.find(item.status!.conditions!, { type: "PIDPressure" });
          const readyCondition = _.find(item.status!.conditions!, { type: "Ready" });

          const conditions = {
            memoryPressure: memoryPressureCondition ? memoryPressureCondition.status === "True" : false,
            diskPressure: diskPressureCondition ? diskPressureCondition.status === "True" : false,
            pidPressure: pidPressureCondition ? pidPressureCondition.status === "True" : false,
            ready: readyCondition ? readyCondition.status === "True" : false,
          };

          return {
            name: item.metadata!.name,
            isConnected: true,
            kubeletVersion: item.status!.nodeInfo!.kubeletVersion,
            cpu: {
              capacity: item.status!.capacity!.cpu,
              allocatable: item.status!.allocatable!.cpu,
            },
            memory: {
              capacity: item.status!.capacity!.memory,
              allocatable: item.status!.allocatable!.memory,
            },
            pods: {
              capacity: item.status!.capacity!.pods,
              allocatable: item.status!.allocatable!.pods,
            },
            conditions,
          };
        });

        return {
          addNodeCommand: "[coming soon]",
          nodes,
        };
      } catch (err) {
        logger.error(err);
        return {
          addNodeCommand: "[unable to show]",
          nodes: [],
        }
      }
    }
  }
}
