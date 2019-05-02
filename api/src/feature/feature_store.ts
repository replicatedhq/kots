import * as jaeger from "jaeger-client";
import * as _ from "lodash";
import { instrumented } from "monkit";
import { Service } from "ts-express-decorators";
import { Params } from "../server/params";
import { traced } from "../server/tracing";
import { PostgresWrapper } from "../util/persistence/db";
import { Feature } from "../generated/types";

@Service()
export class FeatureStore {
  constructor(private readonly wrapper: PostgresWrapper, private readonly params: Params) {}

  @instrumented()
  @traced({ paramTags: { userId: 1 } })
  async listUserFeatures(ctx: jaeger.SpanContext, userId: string): Promise<Feature[]> {
    const q = `
      select f.id from feature f inner join user_feature uf on uf.feature_id = f.id where uf.user_id = $1
    `;
    const v = [userId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    const features: Feature[] = [];
    for (const row of rows) {
      const result = this.mapFeature(row);
      features.push(result);
    }

    return features;
  }

  @instrumented()
  @traced({ paramTags: { watchId: 1 } })
  async listWatchFeatures(ctx: jaeger.SpanContext, watchId: string): Promise<Feature[]> {
    const q = `
      select f.id from feature f inner join watch_feature wf on wf.feature_id = f.id where wf.watch_id = $1
    `;
    const v = [watchId];

    const { rows }: { rows: any[] } = await this.wrapper.query(q, v);
    const features: Feature[] = [];
    for (const row of rows) {
      const result = this.mapFeature(row);
      features.push(result);
    }

    return features;
  }

  private mapFeature(row: any): any {
    return {
      id: row.id,
    };
  }

}
