import _ from "lodash";

export class Analyzer {
  public spec: String;
}


export async function injectKotsAnalyzers(parsedSpec: any): Promise<any> {
  let spec = parsedSpec;

  spec = await injectKubernetesVersionAnalyzer(spec);
  spec = await injectAPIReplicasAnalyzer(spec);
  return spec;
}

async function injectKubernetesVersionAnalyzer(parsedSpec: any): Promise<any> {
  const newAnalyzer = {
    clusterVersion: {
      outcomes: [{
        fail: {
          when: "< 1.13.0",
          message: "The Admin Console requires at least Kubernetes 1.13.0, and recommends 1.15.0",
        },
        warn: {
          when: "< 1.15.0",
          message: "Your cluster meets the minimum required version of Kubernetes, but we recommend using 1.15.0 or later",
        },
        pass: {
          when: ">= 1.15.0",
          message: "Your cluster meets the recommended and required versions of Kubernetes",
        },
      }],
    },
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
      namespace: "test",
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
