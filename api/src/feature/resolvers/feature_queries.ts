import * as _ from "lodash";
import { Context } from "../../context";
import { tracer } from "../../server/tracing";
import { Feature } from "../../generated/types";

export function FeatureQueries(stores: any) {
  return {
    async userFeatures(root: any, args: any, context: Context): Promise<Feature[]> {
      const span = tracer().startSpan("query.listUserFeatures");

      const features = await stores.featureStore.listUserFeatures(span.context(), context.session.userId);
      const result = features.map(feature => toSchemaFeature(feature, root, context));

      span.finish();

      return result;
    },
  }
}

function toSchemaFeature(feature: Feature, root: any, ctx: Context): any {
  return {
    ...feature,
  };
}
