import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { ImageWatchItem, ImageWatchItemsQueryArgs, UploadImageWatchBatchMutationArgs } from "../generated/types";
import { Mutation, Query } from "../schema/decorators";
import { Context } from "../context";
import { tracer } from "../server/tracing";
import { ImageWatchStore } from "./store";

@Service()
export class ImageWatch {
  constructor(private readonly imageWatchStore: ImageWatchStore) {}

  @Mutation("ship-cloud")
  @instrumented({ tags: ["tier:resolver"] })
  async uploadImageWatchBatch(root: any, args: UploadImageWatchBatchMutationArgs, context: Context): Promise<string> {
    const span = tracer().startSpan("mutation.uploadImageWatchBatch");

    const { imageList } = args;

    const batchId = await this.imageWatchStore.createBatch(span.context(), context.session.userId, imageList);

    span.finish();

    return batchId;
  }

  @Query("ship-cloud")
  @instrumented({ tags: ["tier:resolver"] })
  async imageWatchItems(root: any, args: ImageWatchItemsQueryArgs, context: Context): Promise<ImageWatchItem[]> {
    const span = tracer().startSpan("mutation.uploadImageWatchBatch");

    // TODO ownership

    const { batchId } = args;

    const items = await this.imageWatchStore.listImageWatchItemsInBatch(span.context(), batchId);

    span.finish();

    return items;
  }
}
