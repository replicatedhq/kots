import { ValidateUpstreamUrlQueryArgs } from "../../generated/types";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";
import * as jaeger from "jaeger-client";
import { Params } from "../../server/params";
import * as _ from "lodash";
import { WatchStore } from "../../watch/watch_store";

export function InitQueries(stores: any) {
  return {
    async validateUpstreamURL(root: any, { upstream }: ValidateUpstreamUrlQueryArgs, context: Context): Promise<boolean> {
      const span = tracer().startSpan("query.validateUpstreamURL");

      upstream = await maybeRewriteUpstreamUri(stores.watchStore, context.session.userId, upstream);

      // TODO validate this returns a non-error code

      span.finish();

      return true;
    }
  }
}

async function maybeRewriteUpstreamUri(watchStore: WatchStore, userId: string, upstreamUri: string): Promise<string> {
  const span: jaeger.SpanContext = tracer().startSpan("initResolvers.maybeRewriteUpstreamUri");

  if (upstreamUri.startsWith("ship://")) {
    // early version of ships from ships from ships
    // allows for ship://<ship-api-server>/slug
    // and this will rewrite that to be a valid, authenticated api
    // read: https://groups.google.com/forum/#!topic/replicated-engineering/W7QPUMqYcGo

    const match = upstreamUri.match(/^[^:]*:\/\/([^\/]+)(\/.*)$/);

    if (!match) {
      return upstreamUri;
    }
    const slug = _.trimStart(match[2], "/");
    const upstreamWatch = await watchStore.findUserWatch(span.context(), userId, {slug})
    if (!upstreamWatch) {
      return upstreamUri; // don't rewrite if there's no access
    }

    const token = await watchStore.createDownstreamToken(span.context(), upstreamWatch.id!);

    const params = await Params.getParams();
    return `${params.shipApiEndpoint}/api/v1/watch/${upstreamWatch.id}/upstream.yaml?token=${token}`;
  }

  return upstreamUri;
}
