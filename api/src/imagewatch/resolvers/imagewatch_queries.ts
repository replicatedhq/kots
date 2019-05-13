import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function ImageWatchQueries(stores: Stores) {
  return {
    async imageWatches(root: any, args: any, context: Context) {
      // TODO ownership

      const items = await stores.imageWatchStore.listImageWatchesInBatch(args.batchId);

      return items;
    }
  }
}
