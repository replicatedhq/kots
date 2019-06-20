import { Collector, Analyzer } from "./";
import { Params } from "../server/params";
import * as pg from "pg";

export class TroubleshootStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params
  ) {
  }

  public async getPreferedWatchCollector(watchId: string): Promise<Collector> {
    const q = `select release_collector, updated_collector, release_collector_updated_at, updated_collector_updated_at, use_updated_collector
    from watch_troubleshoot_collector where watch_id = $1`;
    const v = [watchId];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      return this.getDefaultCollector();
    }

    const row = result.rows[0];

    let releaseIsNewer: boolean = false;
    if (row.updated_collector_updated_at) {
      releaseIsNewer = new Date(row.updated_collector_updated_at) < new Date(row.release_collector_updated_at);
    };

    let collector: Collector = new Collector();
    if (row.use_updated_collector || !releaseIsNewer) {
      collector.spec = row.updated_collector;
    } else {
      collector.spec = row.release_collector;
    }

    return collector;
  }

  public getDefaultCollector(): Collector {
    const collector: Collector = new Collector();
    collector.spec = `TODO`;

    return collector;
  }
}
