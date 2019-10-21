import _ from "lodash";
import { Params } from "../server/params";
import { parse } from "pg-connection-string"

export class Collector {
  public spec: String;
}

export async function injectKotsCollectors(parsedSpec: any): Promise<any> {
  const uri = (await Params.getParams()).postgresUri;
  const pgConfig = parse(uri);

  let collectorNameBase = "kotsadm-postgres-db";
  const pgDumpCollector = {
    exec: {
      collectorName: collectorNameBase,
      selector: [`app=${pgConfig.host}`],
      containerName: pgConfig.host,
      namespace: process.env["POD_NAMESPACE"],
      command: ["pg_dump"],
      args: ["-U", pgConfig.user],
      timeout: "10s",
    },
  };

  let collectors = _.get(parsedSpec, "spec.collectors") as any[];
  if (!collectors) {
    collectors = [];
  }

  let nameCounter = 1;
  for (let i = 0; i < collectors.length; i++) {
    const collector = collectors[i];
    const name = _.get(collector, "exec.collectorName");
    if (!name) {
      continue;
    }
    if (name === pgDumpCollector.exec.collectorName) {
      pgDumpCollector.exec.collectorName = `${collectorNameBase}_${nameCounter}`;
      nameCounter++;
      i = 0;
      continue;
    }
  }

  collectors.push(pgDumpCollector);
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}
