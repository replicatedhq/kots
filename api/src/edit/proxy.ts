import { IncomingMessage } from "http";
import * as proxyMiddleware from "http-proxy-middleware";
import { parse } from "url";
import { logger } from "../server/logger";

export const EditProxy = proxyMiddleware({
  target: "Ship Cloud Binary - Edit",
  proxyTimeout: 5000,
  router(req: IncomingMessage): string {
    if (!req.url) {
      throw new Error("Invalid URL");
    }
    const { pathname = "" } = parse(req.url);
    const splitPath = pathname.split("/");
    const [, , , , editId] = splitPath;

    // tslint:disable-next-line:no-http-string
    const shipEditInstanceHost = `http://shipedit-${editId}.shipedit-${editId}.svc.cluster.local:8800`;
    logger.debug("proxy path", { shipEditInstanceHost, path: pathname });

    return shipEditInstanceHost;
  },
  pathRewrite(path: string): string {
    const shipEditAPIPath = path.replace(/^\/api\/v1\/edit\/[\w\d]+/, "");
    logger.debug("rewritten Ship Edit API path", { shipEditAPIPath });

    return shipEditAPIPath;
  },
});
