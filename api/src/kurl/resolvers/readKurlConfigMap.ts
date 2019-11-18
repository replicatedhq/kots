import http from "http";
import {
  CoreV1Api,
  KubeConfig,
  V1ConfigMap,
} from "@kubernetes/client-node";
import { ReplicatedError } from "../../server/errors";

export async function readKurlConfigMap(): Promise<{ [ key: string]: string }> {
  const kc = new KubeConfig();
  kc.loadFromDefault();

  const coreV1Client: CoreV1Api = kc.makeApiClient(CoreV1Api);

  let response: http.IncomingMessage;
  let configMap: V1ConfigMap;

  try {
    ({ response, body: configMap } = await coreV1Client.readNamespacedConfigMap("kurl-config", "kube-system"));
  } catch (err) {
    throw new ReplicatedError(`Failed to read config map ${err.response && err.response.body ? err.response.body.message : ""}`);
  }

  if (response.statusCode !== 200 || !configMap) {
    throw new ReplicatedError(`Config map not found`);
  }

  if (!configMap.data) {
    throw new ReplicatedError("Config map data not found");
  }

  return configMap.data;
}
