import * as _ from "lodash";
import { graphiqlExpress, graphqlExpress } from "apollo-server-express";
import bugsnagExpress from "@bugsnag/plugin-express";
import cors from "cors";
import { NextFunction, Request, Response } from "express";
import path from "path";
import Sigsci from "sigsci-module-nodejs";
import { ServerLoader, ServerSettings } from "@tsed/common";
import "@tsed/socketio";
import { $log } from "ts-log-debug";
import { createBugsnagClient } from "./bugsnagClient";
import { ShipClusterSchema } from "../schema";
import { logger } from "./logger";
import { Context } from "../context";
import { getPostgresPool } from "../util/persistence/db";
import { Params } from "./params";
import { UserStore } from "../user/user_store";
import { Stores } from "../schema/stores";
import { SessionStore } from "../session";
import { ClusterStore } from "../cluster";
import { FeatureStore } from "../feature/feature_store";
import { GithubNonceStore } from "../user/store";
import { HealthzStore } from "../healthz/healthz_store";
import { HelmChartStore } from "../helmchart";
import { SnapshotsStore } from "../snapshots";
import { TroubleshootStore } from "../troubleshoot";
import { KotsLicenseStore } from "../klicenses";
import { PreflightStore } from "../preflight/preflight_store";
import { KotsAppStore } from "../kots_app/kots_app_store";
import { KotsAppStatusStore } from "../kots_app/kots_app_status_store";
import tmp from "tmp";
import fs from "fs";
import { KurlStore } from "../kurl/kurl_store";
import { ReplicatedError } from "./errors";
import { MetricStore } from "../monitoring/metric_store";
import { ParamsStore } from "../params/params_store";
import { ensureBucket } from "../util/s3";

let mount = {
  "/": "${rootDir}/../controllers/{*.*s,!(ship)/*.*s}"
};
let componentsScan = [
  "${rootDir}/../middlewares/**/*.ts",
  "${rootDir}/../sockets/kots/*.ts",
];

@ServerSettings({
  rootDir: path.resolve(__dirname),
  httpPort: 3000,
  httpsPort: false,
  mount,
  componentsScan,
  acceptMimes: ["application/json"],
  logger: {
    logRequest: false,
  },
  multer: {
    dest: "${rootDir}/uploads"
  },
  socketIO: {},
})
export class Server extends ServerLoader {
  async $onMountingMiddlewares(): Promise<void> {
    this.expressApp.enable("trust proxy"); // so we get the real ip from the ELB in amazon
    const params = await Params.getParams();

    let bugsnagClient = createBugsnagClient({
      apiKey: params.bugsnagKey,
      appType: "web_server",
      releaseStage: process.env.NODE_ENV
    });

    if (bugsnagClient) {
      bugsnagClient.use(bugsnagExpress);
      const bugsnagMiddleware = bugsnagClient.getPlugin("express");
      this.use(bugsnagMiddleware.requestHandler);
    }

    const corsHeaders = { exposedHeaders: ["Content-Disposition"] };
    this.use(cors(corsHeaders));

    if (params.sigsciRpcAddress) {
      const sigsciOptions = {
        path: params.sigsciRpcAddress,
      };
      const sigsci = new Sigsci(sigsciOptions);
      this.use(sigsci.express());
    }

    this.use(async (req, res, next) => {
      if (req.method === "PUT" && req.url.startsWith("/api/v1/troubleshoot/")) {
        const tmpFile = tmp.fileSync();
        req.on("data", (chunk) => {
          fs.appendFileSync(tmpFile.name, chunk);
        });
        req.on("end", () => {
          req.app.locals.bundleFile = tmpFile.name;
          next();
        });
      } else {
        next();
      }
    });

    const bodyParser = require("body-parser");
    this.use(bodyParser.json({ limit: "5mb" }));

    const pool = await getPostgresPool();
    const paramsStore = new ParamsStore(pool, params);
    const stores: Stores = {
      sessionStore: new SessionStore(pool, params),
      userStore: new UserStore(pool),
      githubNonceStore: new GithubNonceStore(pool),
      clusterStore: new ClusterStore(pool, params),
      featureStore: new FeatureStore(pool, params),
      healthzStore: new HealthzStore(pool, params),
      helmChartStore: new HelmChartStore(pool),
      snapshotsStore: new SnapshotsStore(pool, params),
      troubleshootStore: new TroubleshootStore(pool, params),
      kotsLicenseStore: new KotsLicenseStore(pool, params),
      preflightStore: new PreflightStore(pool),
      kotsAppStore: new KotsAppStore(pool, params),
      kotsAppStatusStore: new KotsAppStatusStore(pool, params),
      kurlStore: new KurlStore(pool, params),
      metricStore: new MetricStore(pool, params, paramsStore),
      paramsStore: new ParamsStore(pool, params),
    }

    if (process.env["AUTO_CREATE_CLUSTER"] === "1") {
      logger.info({msg: "ensuring a local cluster exists"});
      if (process.env["AUTO_CREATE_CLUSTER_NAME"] === "" || process.env["AUTO_CREATE_CLUSTER_TOKEN"] === "") {
        logger.error({msg: "you must set AUTO_CREATE_CLUSTER_NAME and AUTO_CREATE_CLUSTER_TOKEN when AUTO_CREATE_CLUSTER is enabled"});
        process.exit(1);
        return;
      }
      const cluster = await stores.clusterStore.maybeGetClusterWithTypeNameAndToken("ship", process.env["AUTO_CREATE_CLUSTER_NAME"]!, process.env["AUTO_CREATE_CLUSTER_TOKEN"]!);
      if (!cluster) {
        await stores.clusterStore.createNewShipCluster(undefined, true, process.env["AUTO_CREATE_CLUSTER_NAME"]!, process.env["AUTO_CREATE_CLUSTER_TOKEN"]);
      }
    } else {
      logger.debug({msg: "not creating local cluster"});
    }

    if (process.env["SHARED_PASSWORD_BCRYPT"]) {
      logger.info({msg: "ensuring that shared admin console password is provisioned"});
      await stores.userStore.createAdminConsolePassword(process.env["SHARED_PASSWORD_BCRYPT"]!);
    }

    const setContext = async (req: Request, res: Response, next: NextFunction) => {
      let token = req.get("Authorization") || "";

      // remove the "bearer", if it has one
      if (token.startsWith("Bearer")) {
        const splitToken = token.split(" ");
        token = splitToken.pop()!;
      }

      const context = await Context.fetch(stores, token);
      res.locals.context = context;

      next();
    };

    const requireContextGraphql = async (req: Request, res: Response, next: NextFunction) => {
      const anonymousOperations = [
        "ping",
        "logout",
      ];

      if (anonymousOperations.includes(req.body.operationName)) {
        next();
        return;
      }

      if (res.locals.context.requireValidSession()) {
        res.status(403).end();
        return;
      }
      next();
    };

    this.expressApp.locals.stores = stores;

    this.use("/graphql", setContext);
    this.use("/graphql", requireContextGraphql);
    this.use("/graphql", graphqlExpress(async (req: Request, res: Response): Promise<any> => {
      const shipClusterSchema: ShipClusterSchema = new ShipClusterSchema();

      return {
        schema: shipClusterSchema.getSchema(stores, params),
        context: res.locals.context,
        cacheControl: true,
        formatError: (error: any) => {
          logger.error({msg: error.message, error, "stack": error.stack});
          return {
            state: error.originalError && error.originalError.state,
            locations: error.locations,
            path: error.path,
            ...ReplicatedError.getDetails(error),
          };
        },
      };
    }));

    this.expressApp.get("/graphiql", graphiqlExpress({ endpointURL: "/graphql" }));
    // this.use((error: ReplicatedError | Error, request: Request, response: Response, next: NextFunction) => {
    //   if (error instanceof ReplicatedError) {
    //     // logger.error({msg: error.message, error, "stack": error.stack});
    //     return response.send(500, { message: error.originalMessage });
    //   }
    //   throw error;
    // });

    if (process.env.NODE_ENV === "production") {
      $log.level = "OFF";
    }

    // The bugsnag error handler has to go in last
    if (bugsnagClient) {
      const bugsnagMiddleware = bugsnagClient.getPlugin('express');
      this.use(bugsnagMiddleware.errorHandler);
    }
  }

  async $onReady() {
    this.expressApp.get("*", (req: Request, res: Response) => res.sendStatus(404));

    if (process.env["S3_BUCKET_NAME"] === "ship-pacts") {
      logger.info({msg: "Not creating bucket because the desired name is ship-pacts. Consider using a different bucket name to make this work."});
    } else {
      logger.info({msg: "Ensuring bucket exists..."});
      const params = await Params.getParams();
      await ensureBucket(params, params.shipOutputBucket);
    }

    logger.info({msg: "Server started..."});
  }
}
