
export class Collector {
  public spec: String;
}

const POD_NAMESPACE_ENV = "POD_NAMESPACE"
const DEV_NAMESPACE_ENV = "DEV_NAMESPACE"

export function getCollectorNamespace(): String {
  if (process.env[DEV_NAMESPACE_ENV]) {
    return String(process.env[DEV_NAMESPACE_ENV]);
  }
  if (process.env[POD_NAMESPACE_ENV]) {
    return String(process.env[POD_NAMESPACE_ENV]);
  }
  return "default";
}
