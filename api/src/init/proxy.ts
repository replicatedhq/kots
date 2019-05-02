import { IncomingMessage } from "http";
import * as proxyMiddleware from "http-proxy-middleware";
import { parse } from "url";
import { logger } from "../server/logger";

export const proxy = proxyMiddleware({
  target: "Ship Cloud Binary - Init",
  proxyTimeout: 5000,
  router(req: IncomingMessage): string {
    if (!req.url) {
      throw new Error("Invalid URL");
    }
    const { pathname = "" } = parse(req.url);
    const splitPath = pathname.split("/");
    const [, , , , initId] = splitPath;

    // tslint:disable-next-line:no-http-string
    const shipInitInstanceHost = `http://shipinit-${initId}.shipinit-${initId}.svc.cluster.local:8800`;
    logger.debug("proxy path", { shipInitInstanceHost, path: pathname });

    return shipInitInstanceHost;
  },
  pathRewrite(path: string): string {
    const shipInitAPIPath = path.replace(/^\/api\/v1\/init\/[\w\d]+/, "");
    logger.debug("rewritten Ship Init API path", { shipInitAPIPath });

    return shipInitAPIPath;
  },
});
