import { IncomingMessage } from "http";
import proxyMiddleware from "http-proxy-middleware";
import { parse } from "url";

export const UpdateProxy = proxyMiddleware({
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

    return shipUpdateInstanceHost;
  },
  pathRewrite(path: string): string {
    const shipUpdateAPIPath = path.replace(/^\/api\/v1\/update\/[\w\d]+/, "");

    return shipUpdateAPIPath;
  },
});
