import * as _ from "lodash";
import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { authorized } from "../user/decorators";
import { Query } from "../schema/decorators";
import { Context } from "../server/server";
import { tracer } from "../server/tracing";
import { FeatureStore } from "./feature_store";
import { Feature } from "../generated/types";

@Service()
export class FeatureResolvers {
  constructor(
    private readonly featureStore: FeatureStore,
  ) {}

  @Query("ship-cloud")
  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async userFeatures(root: any, args: any, context: Context): Promise<Feature[]> {
    const span = tracer().startSpan("query.listUserFeatures");
    span.setTag("userId", context.userId);

    const features = await this.featureStore.listUserFeatures(span.context(), context.userId);
    const result = features.map(feature => this.toSchemaFeature(feature, root, context));

    span.finish();

    return result;
  }

  @authorized()
  @instrumented({ tags: ["tier:resolver"] })
  async watchFeatures(root: any, args: any, context: Context): Promise<Feature[]> {
    const span = tracer().startSpan("query.listWatchFeatures");

    const { watchId } = args;

    span.setTag("watchId", watchId);

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
