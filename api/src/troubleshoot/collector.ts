import _ from "lodash";
import { Params } from "../server/params";
import { parse } from "pg-connection-string"

export class Collector {
  public spec: String;
}

export function injectKotsCollectors(params: Params, parsedSpec: any, licenseData: string): any {
  let spec = parsedSpec;
  spec = injectDBCollector(params, spec);
  spec = injectLicenseCollector(spec, licenseData);
  spec = injectAPICollector(spec);
  spec = injectOperatorCollector(spec);
  if (params.enableKurl) {
    spec = injectRookCollectors(spec);
    spec = injectKurlCollectors(spec);
  }
  return spec;
}

function injectDBCollector(params: Params, parsedSpec: any): any {
  const uri = params.postgresUri;
  const pgConfig = parse(uri);

  let collectorNameBase = "kotsadm-postgres-db";
  const pgDumpCollector = {
    exec: {
      collectorName: collectorNameBase,
      selector: [`app=${pgConfig.host}`],
      containerName: pgConfig.host,
      namespace: process.env["POD_NAMESPACE"],
      name: "kots/admin_console",
      command: ["pg_dump"],
      args: ["-U", pgConfig.user],
      timeout: "10s",
    },
  };

  let collectors = _.get(parsedSpec, "spec.collectors", []) as any[];

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

function injectLicenseCollector(parsedSpec: any, licenseData: string): any {
  if (!licenseData) {
    return parsedSpec;
  }

  const newCollector = {
    data: {
      collectorName: "license.yaml",
      name: "kots/admin_console",
      data: licenseData,
    },
  };

  let collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    [newCollector],
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectAPICollector(parsedSpec: any): any {
  const newCollector = {
    logs: {
      collectorName: "kotsadm-api",
      selector: ["app=kotsadm-api"],
      namespace: process.env["POD_NAMESPACE"],
      name: "kots/admin_console",
    },
  };

  let collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    [newCollector],
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectOperatorCollector(parsedSpec: any): any {
  const newCollector = {
    logs: {
      collectorName: "kotsadm-operator",
      selector: ["app=kotsadm-operator"],
      namespace: process.env["POD_NAMESPACE"],
      name: "kots/admin_console",
    },
  };

  let collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    [newCollector],
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectRookCollectors(parsedSpec: any): any {
  const names: string[] = [
    "rook-ceph-agent",
    "rook-ceph-mgr",
    "rook-ceph-mon",
    "rook-ceph-operator",
    "rook-ceph-osd",
    "rook-ceph-osd-prepare",
    "rook-ceph-rgw",
    "rook-discover",
  ];
  const newCollectors = _.map(names, (name) => {
    return {
      logs: {
        collectorName: name,
        selector: [`app=${name}`],
        namespace: "rook-ceph",
        name: "kots/rook",
      },
    };
  });

  let collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    newCollectors,
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectKurlCollectors(parsedSpec: any): any {
  const names: string[] = [
    "registry",
  ];
  const newCollectors = _.map(names, (name) => {
    return {
      logs: {
        collectorName: name,
        selector: [`app=${name}`],
        namespace: "kurl",
        name: "kots/kurl",
      },
    };
  });

  let collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    newCollectors,
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

export async function setKotsCollectorsNamespaces(parsedSpec: any): Promise<any> {
  let collectors = _.get(parsedSpec, "spec.collectors") as any[];
  if (!collectors) {
    return parsedSpec;
  }

  for (let collector of collectors) {
    let secret = _.get(collector, "secret");
    if (secret) {
      _.set(secret, "namespace", getCollectorNamespace());
      continue;
    }

    let run = _.get(collector, "run");
    if (run) {
      _.set(run, "namespace", getCollectorNamespace());
      continue;
    }

    let logs = _.get(collector, "logs");
    if (logs) {
      _.set(logs, "namespace", getCollectorNamespace());
      continue;
    }

    let exec = _.get(collector, "exec");
    if (exec) {
      _.set(exec, "namespace", getCollectorNamespace());
      continue;
    }

    let copy = _.get(collector, "copy");
    if (copy) {
      _.set(copy, "namespace", getCollectorNamespace());
      continue;
    }
  }

  return parsedSpec;
}

export function getCollectorNamespace(): String {
  if (process.env["DEV_NAMESPACE"]) {
    return String(process.env["DEV_NAMESPACE"]);
  }
  if (process.env["POD_NAMESPACE"]) {
    return String(process.env["POD_NAMESPACE"]);
  }
  return "default";
}
