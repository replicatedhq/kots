import _ from "lodash";
import { Params } from "../server/params";
import pg from "pg";
import { Feature } from "./";

export class FeatureStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async listUserFeatures(userId: string): Promise<Feature[]> {
    const q = `select f.id from feature f inner join user_feature uf on uf.feature_id = f.id where uf.user_id = $1`;
    const v = [userId];

    const { rows }: { rows: any[] } = await this.pool.query(q, v);
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
