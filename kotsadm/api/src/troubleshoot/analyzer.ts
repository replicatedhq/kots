import _ from "lodash";
import { getCollectorNamespace } from "./collector";

export class Analyzer {
  public spec: String;
}


export async function injectKotsAnalyzers(parsedSpec: any): Promise<any> {
  let spec = parsedSpec;

  if (!spec) {
    spec = {
      apiVersion: "troubleshoot.replicated.com/v1beta1",
      kind: "Analyzer",
      metadata: {
        name: "default-analyzers",
      },
      spec: {
        analyzers: [],
      },
    };
  }

  spec = await injectAPIReplicasAnalyzer(spec);
  spec = await injectOperatorReplicasAnalyzer(spec);
  spec = await injectNoGvisorAnalyzer(spec);

  spec = await injectIfMissingKubernetesVersionAnalyzer(spec);

  return spec;
}

async function injectIfMissingKubernetesVersionAnalyzer(parsedSpec: any): Promise<any> {
  let analyzers = _.get(parsedSpec, "spec.analyzers") as any[];
  if (!analyzers) {
    analyzers = [];
  }

  const currentClusterVersion = _.find(analyzers, (analyzer) => {
    if (analyzer.clusterVersion) {
      return true;
    }

    return false;
  });

  if (!currentClusterVersion) {
    const clusterVersion = {
      clusterVersion: {
        outcomes: [{
          fail: {
            when: "< 1.14.0",
            message: "The Admin Console requires at least Kubernetes 1.14.0, and recommends 1.16.0",
          },
        }, {
          warn: {
            when: "< 1.16.0",
            message: "Your cluster meets the minimum required version of Kubernetes, but we recommend using 1.15.0 or later",
          },
        }, {
          pass: {
            when: ">= 1.16.0",
            message: "Your cluster meets the recommended and required versions of Kubernetes",
          },
        }],
      },
    };

    analyzers.push(clusterVersion);
    _.set(parsedSpec, "spec.analyzers", analyzers);
  }

  return parsedSpec;
}

async function injectNoGvisorAnalyzer(parsedSpec: any): Promise<any> {
  const newAnalyzer = {
    containerRuntime: {
      outcomes: [{
        fail: {
          when: "== gvisor",
          message: "The Admin Console does not support using the gvisor runtime",
        }},{
        pass: {
          message: "A supported container runtime is present on all nodes",
        },
      }],
    }
  };

  let analyzers = _.get(parsedSpec, "spec.analyzers") as any[];
  if (!analyzers) {
    analyzers = [];
  }

  analyzers.push(newAnalyzer);
  _.set(parsedSpec, "spec.analyzers", analyzers);

  return parsedSpec;
}

async function injectOperatorReplicasAnalyzer(parsedSpec: any): Promise<any> {
  const newAnalyzer = {
    deploymentStatus: {
      name: "kotsadm-operator",
      namespace: getCollectorNamespace(),
      outcomes: [{
        pass: {
          when: "= 1",
          message: "Exactly 1 replica of the Admin Console Operator is running and ready",
        }}, {
        fail: {
          message: "There is not exactly 1 replica of the Admin Console Operator running and ready",
        },
      }],
    }
  };

  let analyzers = _.get(parsedSpec, "spec.analyzers") as any[];
  if (!analyzers) {
    analyzers = [];
  }

  analyzers.push(newAnalyzer);
  _.set(parsedSpec, "spec.analyzers", analyzers);

  return parsedSpec;
}

async function injectAPIReplicasAnalyzer(parsedSpec: any): Promise<any> {
  const newAnalyzer = {
    deploymentStatus: {
      name: "kotsadm-api",
      namespace: getCollectorNamespace(),
      outcomes: [{
        pass: {
          when: "> 1",
          message: "At least 2 replicas of the Admin Console API is running and ready",
        },
      },{
        warn: {
          when: "= 1",
          message: "Only 1 replica of the Admin Console API is running and ready",
        },
      },{
        pass: {
          when: "= 0",
          message: "There are no replicas of the Admin Console API running and ready",
        },
      }],
    }
  };

  let analyzers = _.get(parsedSpec, "spec.analyzers") as any[];
  if (!analyzers) {
    analyzers = [];
  }

  analyzers.push(newAnalyzer);
  _.set(parsedSpec, "spec.analyzers", analyzers);

  return parsedSpec;
}
