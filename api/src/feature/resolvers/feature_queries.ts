import * as _ from "lodash";
import { Context } from "../../context";

export function FeatureQueries(stores: any) {
  return {
    async userFeatures(root: any, args: any, context: Context): Promise<any[]> {
      const features = await stores.featureStore.listUserFeatures(context.session.userId);
      const result = features.map((feature) => {
        return {
          id: feature.id,
        };
      });

      return result;
    },
  }
}
