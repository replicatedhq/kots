import pg from "pg";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import { Backup } from "./velero";

export class SnapshotsStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params
  ) {}

  async getKotsBackupSpec(appId: string, sequence: number): Promise<string> {
    const q = `
      SELECT backup_spec FROM app_version WHERE app_id = $1 AND sequence = $2
    `;

    const result = await this.pool.query(q, [appId, sequence]);

    if (result.rows.length === 0) {
      throw new ReplicatedError(`Unable to find Backup Spec with appId ${appId} for sequence ${sequence}`);
    }

    return result.rows[0].backup_spec;
  }
}
