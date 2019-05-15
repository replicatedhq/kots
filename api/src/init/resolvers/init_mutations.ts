import { Context } from "../../context";
import { Params } from "../../server/params";
import * as _ from "lodash";
import { WatchStore } from "../../watch/watch_store";

export function InitMutations(stores: any) {
  return {
    async createInitSession(root: any, { upstreamUri, clusterID, githubPath }: any, context: Context) {
      const uri = await maybeRewriteUpstreamUri(stores.watchStore, context.session.userId, upstreamUri);

      const initSession = await stores.initStore.createInitSession(context.session.userId, uri, clusterID, githubPath, upstreamUri);
      const deployedInitSession = await stores.initStore.deployInitSession(initSession.id);

      return {
        id: deployedInitSession.id,
        upstreamUri: deployedInitSession.upstreamURI,
        createdOn: deployedInitSession.createdOn.toISOString(),
      }
    },
  }
}

async function maybeRewriteUpstreamUri(watchStore: WatchStore, userId: string, upstreamUri: string): Promise<string> {
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
    const upstreamWatch = await watchStore.findUserWatch(userId, {slug})
    if (!upstreamWatch) {
      return upstreamUri; // don't rewrite if there's no access
    }

    const token = await watchStore.createDownstreamToken(upstreamWatch.id!);

    const params = await Params.getParams();
    return `${params.shipApiEndpoint}/api/v1/watch/${upstreamWatch.id}/upstream.yaml?token=${token}`;
  }

  return upstreamUri;
}
