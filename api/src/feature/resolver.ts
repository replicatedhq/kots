import * as _ from "lodash";
import { Service } from "ts-express-decorators";
import { Query } from "../schema/decorators";
import { Context } from "../context";
import { tracer } from "../server/tracing";
import { FeatureStore } from "./feature_store";
import { Feature } from "../generated/types";

@Service()
export class FeatureResolvers {
  constructor(
    private readonly featureStore: FeatureStore,
  ) {}

  @Query("ship-cloud")
  async userFeatures(root: any, args: any, context: Context): Promise<Feature[]> {
    const span = tracer().startSpan("query.listUserFeatures");

    const features = await this.featureStore.listUserFeatures(span.context(), context.session.userId);
    const result = features.map(feature => this.toSchemaFeature(feature, root, context));

    span.finish();

    return result;
  }

  async watchFeatures(root: any, args: any, context: Context): Promise<Feature[]> {
    const span = tracer().startSpan("query.listWatchFeatures");

    const { watchId } = args;

    const features = await this.featureStore.listWatchFeatures(span.context(), watchId);
    const result = features.map(feature => this.toSchemaFeature(feature, root, context));

    span.finish();

    return result;
  }

  private toSchemaFeature(feature: Feature, root: any, ctx: Context): any {
    return {
      ...feature,
    };
  }
}
