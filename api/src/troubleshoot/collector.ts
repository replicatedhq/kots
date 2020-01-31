import _ from "lodash";
import { Params } from "../server/params";
import { parse } from "pg-connection-string";

export class Collector {
  public spec: String;
}

const POD_NAMESPACE_ENV = "POD_NAMESPACE"
const DEV_NAMESPACE_ENV = "DEV_NAMESPACE"

export function injectKotsCollectors(params: Params, parsedSpec: any, licenseData: string): any {
  let spec = parsedSpec;
  spec = injectDBCollector(params, spec);
  spec = injectDBLogsCollector(spec);
  spec = injectLicenseCollector(spec, licenseData);
  spec = injectKotsadmCollector(spec);
  spec = injectAPICollector(spec);
  spec = injectOperatorCollector(spec);
  spec = injectReplicatedPullSecretCollector(spec);
  if (params.enableKurl) {
    spec = injectRookCollectors(spec);
    spec = injectKurlCollectors(spec);
  }
  return spec;
}

function injectDBCollector(params: Params, parsedSpec: any): any {
  const uri = params.postgresUri;
  const pgConfig = parse(uri);

  const collectorNameBase = "kotsadm-postgres-db";
  const pgDumpCollector = {
    exec: {
      collectorName: collectorNameBase,
      selector: [`app=${pgConfig.host}`],
      containerName: pgConfig.host,
      namespace: process.env[POD_NAMESPACE_ENV],
      name: "kots/admin_console",
      command: ["pg_dump"],
      args: ["-U", pgConfig.user],
      timeout: "10s",
    },
  };

  const collectors = _.get(parsedSpec, "spec.collectors", []) as any[];

  let nameCounter = 1;
  for (let i = 0; i < collectors.length; i+=1) {
    const collector = collectors[i];
    const name = _.get(collector, "exec.collectorName");
    if (!name) {
      continue;
    }
    if (name === pgDumpCollector.exec.collectorName) {
      pgDumpCollector.exec.collectorName = `${collectorNameBase}_${nameCounter}`;
      nameCounter+=1;
      i = 0;
      continue;
    }
  }

  collectors.push(pgDumpCollector);
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectDBLogsCollector(parsedSpec: any): any {
  const newCollector = {
    logs: {
      collectorName: "kotsadm-postgres",
      selector: ["app=kotsadm-postgres"],
      namespace: process.env[POD_NAMESPACE_ENV],
      name: "kots/admin_console",
    },
  };

  const collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    [newCollector],
  );
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

  const collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    [newCollector],
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectKotsadmCollector(parsedSpec: any): any {
  const newCollector = {
    logs: {
      collectorName: "kotsadm",
      selector: ["app=kotsadm"],
      namespace: process.env[POD_NAMESPACE_ENV],
      name: "kots/admin_console",
    },
  };

  const collectors = _.concat(
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
      namespace: process.env[POD_NAMESPACE_ENV],
      name: "kots/admin_console",
    },
  };

  const collectors = _.concat(
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
      namespace: process.env[POD_NAMESPACE_ENV],
      name: "kots/admin_console",
    },
  };

  const collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    [newCollector],
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

function injectReplicatedPullSecretCollector(parsedSpec: any): any {
  const newCollector = {
    secret: {
      collectorName: "kotsadm-replicated-registry",
      namespace: getCollectorNamespace(),
      name: "kotsadm-replicated-registry",
      key: ".dockerconfigjson",
      includeValue: false,
    },
  };

  const collectors = _.concat(
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

  const collectors = _.concat(
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

  const collectors = _.concat(
    _.get(parsedSpec, "spec.collectors", []) as any[],
    newCollectors,
  );
  _.set(parsedSpec, "spec.collectors", collectors);

  return parsedSpec;
}

export async function setKotsCollectorsNamespaces(parsedSpec: any): Promise<any> {
  const collectors = _.get(parsedSpec, "spec.collectors") as any[];
  if (!collectors) {
    return parsedSpec;
  }

  for (const collector of collectors) {
    const secret = _.get(collector, "secret");
    if (secret) {
      _.set(secret, "namespace", getCollectorNamespace());
      continue;
    }

    const run = _.get(collector, "run");
    if (run) {
      _.set(run, "namespace", getCollectorNamespace());
      continue;
    }

    const logs = _.get(collector, "logs");
    if (logs) {
      _.set(logs, "namespace", getCollectorNamespace());
      continue;
    }

    const exec = _.get(collector, "exec");
    if (exec) {
      _.set(exec, "namespace", getCollectorNamespace());
      continue;
    }

    const copy = _.get(collector, "copy");
    if (copy) {
      _.set(copy, "namespace", getCollectorNamespace());
      continue;
    }
  }

  return parsedSpec;
}

export function getCollectorNamespace(): String {
  if (process.env[DEV_NAMESPACE_ENV]) {
    return String(process.env[DEV_NAMESPACE_ENV]);
  }
  if (process.env[POD_NAMESPACE_ENV]) {
    return String(process.env[POD_NAMESPACE_ENV]);
  }
  return "default";
}
