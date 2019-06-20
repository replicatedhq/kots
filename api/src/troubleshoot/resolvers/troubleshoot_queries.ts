import { Context } from "../../context";
import { Stores } from "../../schema/stores";
import { ReplicatedError } from "../../server/errors";

export function TroubleshootQueries(stores: Stores) {
  return {
    async watchTroubleshootCollectors(root: any, args: any, context: Context) {
      const watch = await context.getWatch(args.watchId);

      const collector = await stores.troubleshootStore.getPreferedWatchCollector(watch.id);

      return collector.spec;
    }
  }
}
