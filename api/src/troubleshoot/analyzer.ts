import _ from "lodash";

export class Analyzer {
  public spec: String;
}


export async function injectKotsAnalyzers(parsedSpec: any): Promise<any> {
  let spec = parsedSpec;

  spec = await injectAPIReplicasAnalyzer(spec);
  return spec;
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
