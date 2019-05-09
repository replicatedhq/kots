import { ImageWatchItem, UploadImageWatchBatchMutationArgs } from "../../generated/types";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";

export function ImageWatchMutations(stores: any) {
  return {
    async uploadImageWatchBatch(root: any, args: UploadImageWatchBatchMutationArgs, context: Context): Promise<string> {
      const span = tracer().startSpan("mutation.uploadImageWatchBatch");

      const { imageList } = args;

      const batchId = await stores.imageWatchStore.createBatch(span.context(), context.session.userId, imageList);

      span.finish();

      return batchId;
    }
  }
}
