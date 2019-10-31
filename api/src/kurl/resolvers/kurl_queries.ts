import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";
import * as k8s from "@kubernetes/client-node";
import _ from "lodash";
import request from "request-promise";
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
        const nodes = _.map(res.body.items, async (item) => {
          let usageStats = {
            availableCPUs: -1,
            availableMemory: -1,
            availablePods: -1
          };
          const address = _.find(item.status!.addresses || [], { type: "InternalIP" });
          if (address) {
            const nodeIP = address.address;
            const options = {
              method: "GET",
              uri: `http://${nodeIP}:10255/stats/summary`
            };
            const response = await request(options);
            if (response) {
              const stats = JSON.parse(response);
              const totalCPUs = parseFloat(item.status!.capacity!.cpu!);
              usageStats.availableCPUs = totalCPUs - (stats.node!.cpu!.usageNanoCores! / Math.pow(10, 9));
              usageStats.availableMemory = stats.node!.memory!.availableBytes! / 1073741824;
              usageStats.availablePods = parseInt(item.status!.capacity!.pods!) - stats.pods!.length;
            }
          }

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

          let memoryCapacityStr = item.status!.capacity!.memory; // example: 134123213Ki
          memoryCapacityStr = memoryCapacityStr.substring(0, memoryCapacityStr.length - 2);
          const memoryCapacity = parseFloat(memoryCapacityStr) / 976562.5;

          return {
            name: item.metadata!.name,
            isConnected: true,
            // TODO need to check for pods on the node
            canDelete: !!(item.spec!.unschedulable),
            kubeletVersion: item.status!.nodeInfo!.kubeletVersion,
            cpu: {
              capacity: item.status!.capacity!.cpu,
              available: usageStats.availableCPUs,
            },
            memory: {
              capacity: memoryCapacity,
              available: usageStats.availableMemory,
            },
            pods: {
              capacity: item.status!.capacity!.pods,
              available: usageStats.availablePods,
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
