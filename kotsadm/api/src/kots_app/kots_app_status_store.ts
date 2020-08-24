import pg from "pg";
import { Params } from "../server/params";
import { KotsAppStatus } from "./";
import { ReplicatedError } from "../server/errors";
import _ from "lodash";

export class KotsAppStatusStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async getKotsAppStatus(appId: string): Promise<KotsAppStatus> {
    const q = `select resource_states, updated_at from app_status where app_id = $1`;
    const v = [
      appId,
    ];

    const result = await this.pool.query(q, v);

    if (result.rowCount == 0) {
      throw ReplicatedError.notFound();
    }

    const kotsAppStatus = new KotsAppStatus();
    kotsAppStatus.appId = appId;
    kotsAppStatus.updatedAt = result.rows[0].updated_at;
    kotsAppStatus.resourceStates = JSON.parse(result.rows[0].resource_states);

    return kotsAppStatus;
  }
}
