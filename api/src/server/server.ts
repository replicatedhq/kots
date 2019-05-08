import { graphiqlExpress, graphqlExpress } from "apollo-server-express";
import * as bugsnag from "bugsnag";
import * as cors from "cors";
import { NextFunction, Request, Response } from "express";
import * as path from "path";
import * as Sigsci from "sigsci-module-nodejs";
import { ServerLoader, ServerSettings } from "ts-express-decorators";
import { $log } from "ts-log-debug";
import { isPolicyValid } from "../user/policy";
import { proxy as InitProxy } from "../init/proxy";
import { ShipClusterSchema } from "../schema";
import { proxy as UpdateProxy } from "../update/proxy";
import { ReplicatedError } from "./errors";
import { configureInjector } from "./injector";
import { logger } from "./logger";
import { Context } from "../context";

import { PostgresWrapper, getPostgresPool } from "../util/persistence/db";
import { Params } from "./params";
import { UserStore } from "../user/user_store";
import { Stores } from "../schema/stores";
import { SessionStore } from "../session";
import { ClusterStore } from "../cluster";
import { WatchStore } from "../watch/watch_store";
import { NotificationStore } from "../notification/store";
import { UpdateStore } from "../update/store";
import { UnforkStore } from "../unfork/unfork_store";
import { InitStore } from "../init/init_store";
import { ImageWatchStore } from "../imagewatch/store";
import { FeatureStore } from "../feature/feature_store";

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
  async $onInit(): Promise<void> {
    await configureInjector();
  }

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

    const bodyParser = require("body-parser");
    this.use(bodyParser.json());

    const setContext = async (req: Request, res: Response, next: NextFunction) => {
      const token = req.get("Authorization") || "";

      const context = await Context.fetch(token);
      res.locals.context = context;

      next();
    };

    this.use("/api/v1/download/:watchId", setContext);
    this.use("/api/v1/download/:watchId", (request: Request, response: Response, next: NextFunction) => {
      next(isPolicyValid(response.locals.context));
    });

    this.use("/graphql", setContext);
    this.use("/graphql", graphqlExpress(async (req: Request, res: Response): Promise<any> => {
      const wrapper = new PostgresWrapper(await getPostgresPool());
      const params = await Params.getParams();
      const stores: Stores = {
        sessionStore: new SessionStore(wrapper, params),
        userStore: new UserStore(wrapper),
        clsuterStore: new ClusterStore(wrapper, params),
        watchStore: new WatchStore(wrapper, params),
        notificationStore: new NotificationStore(wrapper, params),
        updateStore: new UpdateStore(wrapper, params),
        unforkStore: new UnforkStore(wrapper, params),
        initStore: new InitStore(wrapper, params),
        imageWatchStore: new ImageWatchStore(wrapper),
        featureStore: new FeatureStore(wrapper, params),
      }

      return {
        schema: new ShipClusterSchema().getSchema(stores),
        context: res.locals.context,
        tracing: true,
        cacheControl: true,
        formatError: (error: any) => {
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
    this.use((error: ReplicatedError | Error, request: Request, response: Response, next: NextFunction) => {
      if (error instanceof ReplicatedError) {
        if (error.code) {
          logger.error(error);
          logger.error(error.stack!);
          return response.send(+error.code, { message: error.originalMessage });
        }
      }
      throw error;
    });

    if (process.env.NODE_ENV === "production") {
      $log.level = "OFF";
    }
  }

  $onReady() {
    this.expressApp.get("*", (req: Request, res: Response) => res.sendStatus(404));
    logger.info("Server started...");
  }

  $onServerInitError(err: Error) {
    logger.error(err);
  }
}
