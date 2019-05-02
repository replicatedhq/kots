import { IncomingMessage } from "http";
import * as proxyMiddleware from "http-proxy-middleware";
import { parse } from "url";
import { logger } from "../server/logger";

export const proxy = proxyMiddleware({
  target: "Ship Cloud Binary - Update",
  proxyTimeout: 5000,
  router(req: IncomingMessage): string {
    if (!req.url) {
      throw new Error("Invalid URL");
    }
    const { pathname = "" } = parse(req.url);
    const splitPath = pathname.split("/");
    const [, , , , updateId] = splitPath;

    // tslint:disable-next-line:no-http-string
    const shipUpdateInstanceHost = `http://shipupdate-${updateId}.shipupdate-${updateId}.svc.cluster.local:8800`;
    logger.debug("proxy path", { shipUpdateInstanceHost, path: pathname });

    return shipUpdateInstanceHost;
  },
  pathRewrite(path: string): string {
    const shipUpdateAPIPath = path.replace(/^\/api\/v1\/update\/[\w\d]+/, "");
    logger.debug("rewritten Ship Update API path", { shipUpdateAPIPath });

    return shipUpdateAPIPath;
  },
});
