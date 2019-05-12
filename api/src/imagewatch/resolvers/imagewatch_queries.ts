import { ImageWatchItem, ImageWatchItemsQueryArgs } from "../../generated/types";
import { Context } from "../../context";
import { Stores } from "../../schema/stores";

export function ImageWatchQueries(stores: Stores) {
  return {
    async imageWatchItems(root: any, args: ImageWatchItemsQueryArgs, context: Context): Promise<ImageWatchItem[]> {
      // TODO ownership

      const { batchId } = args;

      const items = await stores.imageWatchStore.listImageWatchItemsInBatch(batchId);

      return items;
    }
  }
}
