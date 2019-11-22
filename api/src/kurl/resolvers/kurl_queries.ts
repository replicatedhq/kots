import * as fs from "fs";
import * as util from "util";
import { Stores } from "../../schema/stores";
import { Params } from "../../server/params";
import { Context } from "../../context";
import * as k8s from "@kubernetes/client-node";
import _ from "lodash";
import request, { RequestPromiseOptions } from "request-promise";
import { logger } from "../../server/logger";
import { readKurlConfigMap } from "./readKurlConfigMap";
import { ReplicatedError } from "../../server/errors";

const readFile = util.promisify(fs.readFile);

export function KurlQueries(stores: Stores, params: Params) {
  return {
    async kurl(root: any, args: any, context: Context): Promise<any> {
      context.requireSingleTenantSession();

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
            try {
              const nodeIP = address.address;
              let uri = `http://${nodeIP}:10255/stats/summary`;
              const options: RequestPromiseOptions = {
                method: "GET",
              };
              try {
                options.key = await readFile("/etc/kubernetes/pki/kubelet/client.key");
                options.cert = await readFile("/etc/kubernetes/pki/kubelet/client.crt");
                // kubelet server cert is self-sigend (/var/lib/kubelet/pki/kubelet.crt)
                options.strictSSL = false;
                uri = `https://${nodeIP}:10250/stats/summary`;
              } catch(err) {
                // ignore, this is for kurl clusters only
              }
              const response = await request(uri, options);
              if (response) {
                const stats = JSON.parse(response);
                const totalCPUs = parseFloat(item.status!.capacity!.cpu!);
                usageStats.availableCPUs = totalCPUs - (stats.node!.cpu!.usageNanoCores! / Math.pow(10, 9));
                usageStats.availableMemory = stats.node!.memory!.availableBytes! / 1073741824;
                usageStats.availablePods = parseInt(item.status!.capacity!.pods!) - stats.pods!.length;
              }
            } catch(err) {
              console.log(`Failed to read node stats: ${err}`);
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

        let isKurlEnabled = false;
        let ha = false;
        try {
          const data = await readKurlConfigMap();
          isKurlEnabled = true;
          ha = !!data.ha;
        } catch(err) {
          // this is expected if no kurl
          if (!(err instanceof ReplicatedError) || err.originalMessage !== "Config map not found") {
            console.log(err);
          }
        }

        return {
          nodes,
          isKurlEnabled,
          ha: ha,
        };
      } catch (err) {
        logger.error(err);
        return {
          nodes: [],
          ha: false,
        }
      }
    }
  }
}
