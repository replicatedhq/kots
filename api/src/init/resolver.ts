import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { authorized } from "../auth/decorators";
import { CreateInitSessionMutationArgs, InitSession, ValidateUpstreamUrlQueryArgs } from "../generated/types";
import { Mutation, Query } from "../schema/decorators";
import { Context } from "../server/server";
import { tracer } from "../server/tracing";
import { InitStore } from "./init_store";
import { WatchStore } from "../watch/watch_store";
import * as jaeger from "jaeger-client";
import { Params } from "../server/params";
import * as _ from "lodash";

@Service()
export class Init {
  constructor(
    private readonly initStore: InitStore,
    private readonly watchStore: WatchStore,
  ) {
  }

  @Mutation("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async createInitSession(root: any, { upstreamUri, clusterID, githubPath }: CreateInitSessionMutationArgs, context: Context): Promise<InitSession> {
    const span = tracer().startSpan("mutation.createInitSession");

    const uri = await this.maybeRewriteUpstreamUri(span.context(), context.userId, upstreamUri);

    const initSession = await this.initStore.createInitSession(span.context(), context.userId, uri, clusterID, githubPath, upstreamUri);
    const deployedInitSession = await this.initStore.deployInitSession(span.context(), initSession.id!);

    span.finish();

    return deployedInitSession;
  }

  @instrumented()
  @Query("ship-cloud")
  @authorized()
  async validateUpstreamURL(root: any, { upstream }: ValidateUpstreamUrlQueryArgs, context: Context): Promise<boolean> {
    const span = tracer().startSpan("query.validateUpstreamURL");

    upstream = await this.maybeRewriteUpstreamUri(span.context(), context.userId, upstream);

    // TODO validate this returns a non-error code

    span.finish();

    return true;
  }

  async maybeRewriteUpstreamUri(ctx: jaeger.SpanContext, userId: string, upstreamUri: string): Promise<string> {
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
      const upstreamWatch = await this.watchStore.findUserWatch(span.context(), userId, {slug})
      if (!upstreamWatch) {
        return upstreamUri; // don't rewrite if there's no access
      }

      const token = await this.watchStore.createDownstreamToken(span.context(), upstreamWatch.id!);

      const params = await Params.getParams();
      return `${params.shipApiEndpoint}/api/v1/watch/${upstreamWatch.id}/upstream.yaml?token=${token}`;
    }

    return upstreamUri;
  }
}
