import { DataDogMetricRegistry, getRegistry, Registry, setNamer, setRegistry } from "monkit";
import * as rp from "request-promise";
import { logger } from "./logger";

let statsdOpts: {};
let registry: Registry;

async function statsdOptions(): Promise<any> {
  if (statsdOpts) {
    return statsdOpts;
  }

  let statsdIpAddress;
  if (process.env.USE_EC2_PARAMETERS) {
    const options = {
      // tslint:disable-next-line no-http-string
      uri: "http://169.254.169.254/latest/meta-data/local-ipv4",
    };

    statsdIpAddress = await Promise.resolve(rp(options));
  }

  const statsdHost = statsdIpAddress || process.env.STATSD_HOST;
  const statsdPort = process.env.STATSD_PORT || 8125;
  const statsdIntervalMillis = process.env.STATSD_INTERVAL_MILLIS || 30000;
  const statsdPrefix = process.env.STATSD_PREFIX || "";
  const globalTags = [`service:ship-cluster-api`, "product:ship-cluster"];
  statsdOpts = {
    host: statsdHost,
    port: statsdPort,
    interval: statsdIntervalMillis,
    prefix: statsdPrefix,
    tags: globalTags,
  };

  return statsdOpts;
}

export async function metrics(): Promise<Registry> {
  if (registry) {
    return registry;
  }

  const { host, port, prefix, tags } = await statsdOptions();
  if (!host) {
    logger.error("neither the AWS Metadata Service nor STATSD_HOST is set, default Monkit registry will be used");

    return getRegistry();
  }
  logger.info({ host, port, prefix, tags }, "Initializing statsd metrics");

  registry = DataDogMetricRegistry.hotShots({ host, port, prefix, globalTags: tags }, err => {
    logger.error(err, "failed reporting statsd metrics");
  });

  setRegistry(registry);
  setNamer((k, s) => "instrumented");

  registry.meter("monkit.startup_ping", ["meta"]).mark();

  return registry;
}
