import { graphiqlExpress, graphqlExpress } from "apollo-server-express";
import * as bugsnag from "bugsnag";
import * as cors from "cors";
import { NextFunction, Request, Response } from "express";
import * as path from "path";
import * as Sigsci from "sigsci-module-nodejs";
import { ServerLoader, ServerSettings } from "ts-express-decorators";
import { $log } from "ts-log-debug";
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
import { ImageWatchStore } from "../imagewatch/imagewatch_store";
import { FeatureStore } from "../feature/feature_store";
import { GithubNonceStore } from "../user/store";
import { HealthzStore } from "../healthz/store";
import { WatchDownload } from "../watch/download";
import { EditStore } from "../edit";
import { PendingStore } from "../pending";
import { HelmChartStore } from "../helmchart";
import { TroubleshootStore } from "../troubleshoot";
import { LicenseStore } from "../license";

const tsedConfig = {
  rootDir: path.resolve(__dirname),
  mount: {
    // tslint:disable-next-line
    "/": "${rootDir}/../controllers/**/*.*s",
  },
  acceptMimes: ["application/json"],
  componentsScan: [],
  port: 3000,
  httpsPort: 0,
  debug: false,
};

@ServerSettings(tsedConfig)
export class Server extends ServerLoader {
  async $onMountingMiddlewares(): Promise<void> {
    this.expressApp.enable("trust proxy"); // so we get the real ip from the ELB in amazon
    const params = await Params.getParams();

    if (params.bugsnagKey) {
      bugsnag.register(params.bugsnagKey);
      this.use(bugsnag.errorHandler);
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

    // Place http-proxy-middleware before body-parser
    // See https://github.com/chimurai/http-proxy-middleware/issues/40#issuecomment-163398924
    this.use("/api/v1/init/:id", InitProxy);
    this.use("/api/v1/update/:id", UpdateProxy);
    this.use("/api/v1/edit/:id", EditProxy);

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
      imageWatchStore: new ImageWatchStore(pool),
      featureStore: new FeatureStore(pool, params),
      healthzStore: new HealthzStore(pool),
      watchDownload: new WatchDownload(watchStore),
      editStore: new EditStore(pool, params),
      pendingStore: new PendingStore(pool, params),
      helmChartStore: new HelmChartStore(pool),
      troubleshootStore: new TroubleshootStore(pool, params),
      licenseStore: new LicenseStore(pool, params),
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
        // formatError: (error: any) => {
        //   return {
        //     state: error.originalError && error.originalError.state,
        //     locations: error.locations,
        //     path: error.path,
        //     ...ReplicatedError.getDetails(error),
        //   };
        // },
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
  }

  $onReady() {
    this.expressApp.get("*", (req: Request, res: Response) => res.sendStatus(404));
    logger.info({msg: "Server started..."});
  }

  $onServerInitError(err: Error) {
    logger.error({msg: err.message, err});
  }
}
