import { ImageWatchItem, ImageWatchItemsQueryArgs } from "../../generated/types";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";

export function ImageWatchQueries(stores: any) {
  return {
    async imageWatchItems(root: any, args: ImageWatchItemsQueryArgs, context: Context): Promise<ImageWatchItem[]> {
      const span = tracer().startSpan("mutation.uploadImageWatchBatch");

      // TODO ownership

      const { batchId } = args;

      const items = await stores.imageWatchStore.listImageWatchItemsInBatch(span.context(), batchId);

      span.finish();

      return items;
    }
  }
}
