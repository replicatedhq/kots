import * as AWS from "aws-sdk";
import * as jaeger from "jaeger-client";
import { DataDogMetricRegistry } from "monkit";
import { InjectorService, Service } from "ts-express-decorators";

import { GitHubHookAPI } from "../controllers/GitHubHookAPI";
import { HealthzAPI } from "../controllers/HealthzAPI";
import { Metricz } from "../controllers/MetriczAPI";
import { WatchDownloadAPI } from "../controllers/WatchDownloadAPI";
import { DeployAPI } from "../controllers/DeployAPI";
import { RootAPI } from "../controllers/RootAPI";
import { WatchAPI } from "../controllers/WatchAPI";
import { InstallAPI } from "../controllers/InstallAPI";

import { ReplicatedSchema } from "../schema";
import { Mutation, Query } from "../schema/decorators";
import { DefaultClock } from "../util/clock";
import { logger } from "./logger";

import { Pool } from "pg";
import { Auth } from "../auth";
import { GitHub } from "../github_installation/github";
import { ImageWatch } from "../imagewatch/resolver";
import { Init } from "../init/resolver";
import { Unfork } from "../unfork/resolver";
import { ShipNotification } from "../notification/resolver";
import { Session } from "../session";
import { Update } from "../update/resolver";
import { getPostgresPool, PostgresWrapper } from "../util/persistence/db";
import { Watch } from "../watch/resolver";
import { Cluster } from "../cluster/resolver";
import { metrics } from "./metrics";
import { Params } from "./params";
import { tracer } from "./tracing";

@Service()
export class HealthzResolver {
  @Query("ship-cloud")
  async healthz(): Promise<{}> {
    return {
      version: process.env.VERSION || "unknown",
    };
  }

  @Mutation("ship-cloud")
  async ping(): Promise<string> {
    logger.info("got ping");

    return "pong";
  }
}

const bind = (sym: any) => {
  InjectorService.service(sym);

  return (instance: any) => {
    InjectorService.set(sym, instance);
  };
};

export async function configureInjector(): Promise<void> {
  const pool = await getPostgresPool();
  const tracing = tracer();

  bind(Pool)(pool);
  bind(jaeger.Tracer)(tracing);
  bind(PostgresWrapper)(new PostgresWrapper(pool));
  bind(DefaultClock)(new DefaultClock());
  bind(Params)(await Params.getParams());
  bind(AWS.S3)(new AWS.S3({ apiVersion: "2012-06-01", signatureVersion: 'v4' }));
  bind(DataDogMetricRegistry)(await metrics());
  bind(HealthzAPI);
  bind(Metricz);
  bind(GitHubHookAPI);
  bind(WatchDownloadAPI);
  bind(Auth);
  bind(Watch);
  bind(Cluster);
  bind(ImageWatch);
  bind(Init);
  bind(Unfork);
  bind(Update);
  bind(ShipNotification);
  bind(GitHub);
  bind(Session);
  bind(ReplicatedSchema);
  bind(DeployAPI);
  bind(RootAPI);
  bind(WatchAPI);
  bind(InstallAPI);
}
