import { Context } from "../../context";

export function ImageWatchMutations(stores: any) {
  return {
    async uploadImageWatchBatch(root: any, args: any, context: Context): Promise<string> {
      const batchId = await stores.imageWatchStore.createBatch(context.session.userId, args.imageList);
      return batchId;
    }
  }
}
