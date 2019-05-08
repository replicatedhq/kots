import * as AWS from "aws-sdk";
import * as jaeger from "jaeger-client";
import { DataDogMetricRegistry } from "monkit";
import { InjectorService, Service } from "ts-express-decorators";

import { DefaultClock } from "../util/clock";
import { logger } from "./logger";

import { Pool } from "pg";
import { Session } from "../session";
import { getPostgresPool, PostgresWrapper } from "../util/persistence/db";
import { metrics } from "./metrics";
import { Params } from "./params";
import { tracer } from "./tracing";

@Service()
export class HealthzResolver {
  async healthz(): Promise<{}> {
    return {
      version: process.env.VERSION || "unknown",
    };
  }

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
  bind(Session);
}
