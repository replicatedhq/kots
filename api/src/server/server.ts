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
import { InitProxy } from "../init/proxy";
import { ShipClusterSchema } from "../schema";
import { UpdateProxy } from "../update/proxy";
import { EditProxy } from "../edit/proxy";
import { logger } from "./logger";
import { Context } from "../context";
import { getPostgresPool } from "../util/persistence/db";
import { Params } from "./params";
import { UserStore } from "../user/user_store";
import { Stores } from "../schema/stores";
import { SessionStore } from "../session";
import { ClusterStore } from "../cluster";
import { WatchStore } from "../watch/watch_store";
import { NotificationStore } from "../notification";
import { UpdateStore } from "../update/update_store";
import { UnforkStore } from "../unfork/unfork_store";
import { InitStore } from "../init/init_store";
import { FeatureStore } from "../feature/feature_store";
import { GithubNonceStore } from "../user/store";
import { HealthzStore } from "../healthz/healthz_store";
import { WatchDownload } from "../watch/download";
import { EditStore } from "../edit";
import { PendingStore } from "../pending";
import { HelmChartStore } from "../helmchart";
import { TroubleshootStore } from "../troubleshoot";
import { LicenseStore } from "../license";
import { KotsLicenseStore } from "../klicenses";
import { GithubInstallationsStore } from "../github_installation/github_installation_store";
import { PreflightStore } from "../preflight/preflight_store";
import { KotsAppStore } from "../kots_app/kots_app_store";
import tmp from "tmp";
import fs from "fs";
import { KurlStore } from "../kurl/kurl_store";
import { ReplicatedError } from "./errors";

let mount = {};
let componentsScan = [
  "${rootDir}/../middlewares/**/*.ts",
];

const enableShip = process.env["ENABLE_SHIP"] === "1";
const enableKots = process.env["ENABLE_KOTS"] === "1";
if (enableKots && enableShip) {
  mount = {
    "/": "${rootDir}/../controllers/**/*.*s",
  };
  componentsScan.push("${rootDir}/../sockets/kots/*.ts");
} else if (enableShip) {
  mount = {
    "/": "${rootDir}/../controllers/{*.*s,!(kots)/*.*s}",
  };
} else if (enableKots) {
  mount = {
    "/": "${rootDir}/../controllers/{*.*s,!(ship)/*.*s}",
  };
  componentsScan.push("${rootDir}/../sockets/kots/*.ts");
} else {
  mount = {
    "/": "${rootDir}/../controllers/*.*s",
  };
}

@ServerSettings({
  rootDir: path.resolve(__dirname),
  httpPort: 3000,
  httpsPort: false,
  mount,
  componentsScan,
  acceptMimes: ["application/json"],
  debug: true,
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

    // Place http-proxy-middleware before body-parser
    // See https://github.com/chimurai/http-proxy-middleware/issues/40#issuecomment-163398924
    if (process.env["ENABLE_SHIP"]) {
      this.use("/api/v1/init/:id", InitProxy);
      this.use("/api/v1/update/:id", UpdateProxy);
      this.use("/api/v1/edit/:id", EditProxy);
    }

    const bodyParser = require("body-parser");
    this.use(bodyParser.json({ limit: "5mb" }));

    const pool = await getPostgresPool();
    const watchStore = new WatchStore(pool, params);
    const stores: Stores = {
      sessionStore: new SessionStore(pool, params),
      userStore: new UserStore(pool),
      githubNonceStore: new GithubNonceStore(pool),
      clusterStore: new ClusterStore(pool, params),
      watchStore: watchStore,
      notificationStore: new NotificationStore(pool),
      updateStore: new UpdateStore(pool, params),
      unforkStore: new UnforkStore(pool, params),
      initStore: new InitStore(pool, params),
      featureStore: new FeatureStore(pool, params),
      healthzStore: new HealthzStore(pool, params),
      watchDownload: new WatchDownload(watchStore),
      editStore: new EditStore(pool, params),
      pendingStore: new PendingStore(pool, params),
      helmChartStore: new HelmChartStore(pool),
      troubleshootStore: new TroubleshootStore(pool, params),
      licenseStore: new LicenseStore(pool, params),
      kotsLicenseStore: new KotsLicenseStore(pool, params),
      githubInstall: new GithubInstallationsStore(pool),
      preflightStore: new PreflightStore(pool),
      kotsAppStore: new KotsAppStore(pool, params),
      kurlStore: new KurlStore(pool, params),
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

    if(process.env["SHARED_PASSWORD_BCRYPT"]) {
      logger.info({msg: "ensuring that shared admin console password is provisioned"});
      await stores.userStore.createAdminConsolePassword(process.env["SHARED_PASSWORD_BCRYPT"]!);
    }

    const setContext = async (req: Request, res: Response, next: NextFunction) => {
      const token = req.get("Authorization") || "";

      const context = await Context.fetch(stores, token);
      res.locals.context = context;

      next();
    };

    this.expressApp.locals.stores = stores;

    this.use("/api/v1/download/:watchId", setContext);
    this.use("/api/v1/download/:watchId", (request: Request, response: Response, next: NextFunction) => {
      next(response.locals.context.hasValidSession());
    });

    this.use("/graphql", setContext);
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

  $onReady() {
    this.expressApp.get("*", (req: Request, res: Response) => res.sendStatus(404));
    logger.info({msg: "Server started..."});
  }

  $onServerInitError(err: Error) {
    logger.error({msg: err.message, err});
  }
}
