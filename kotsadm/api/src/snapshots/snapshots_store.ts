import * as _ from "lodash";
import pg from "pg";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import { ScheduledSnapshot } from "./snapshot";
import { Querier } from "../util/persistence/db";

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

  async listPendingScheduledSnapshots(appId: string): Promise<ScheduledSnapshot[]> {
    const q = `
      SELECT id, app_id, scheduled_timestamp FROM scheduled_snapshots WHERE app_id = $1 AND backup_name IS NULL;
    `;

    const result = await this.pool.query(q, [appId]);
 
    return _.map(result.rows, (row) => {
      return {
        appId,
        id: row.id,
        scheduledTimestamp: row.scheduled_timestamp,
      };
    });
  }

  async deletePendingScheduledSnapshots(appId: string, tx?: Querier): Promise<void> {
    const q = `
      DELETE FROM scheduled_snapshots WHERE app_id = $1 AND backup_name IS NULL
    `;

    if (tx) {
      await tx.query(q, [appId]);
    } else {
      await this.pool.query(q, [appId]);
    }
  }

  async lockScheduledSnapshot(tx: Querier, id: string): Promise<boolean> {
    const q = `SELECT * FROM scheduled_snapshots WHERE id = $1 FOR UPDATE SKIP LOCKED`;

    const result = await tx.query(q, [id]);
    if (result.rows.length) {
      return true;
    }
    return false;
  }

  async updateScheduledSnapshot(tx: Querier, id: string, backupName: string) {
    const q = `
      UPDATE scheduled_snapshots SET backup_name = $1 WHERE id = $2
    `;
    const v = [backupName, id];

    await tx.query(q, v);
  }

  async createScheduledSnapshot(ss: ScheduledSnapshot, tx?: Querier): Promise<void> {
    const q = `
      INSERT INTO scheduled_snapshots (
        id,
        app_id,
        scheduled_timestamp
      ) VALUES (
        $1,
        $2,
        $3
      )
    `;
    const v = [ss.id, ss.appId, ss.scheduledTimestamp];

    if (tx) {
      await tx.query(q, v);
    } else {
      await this.pool.query(q, v);
    }
  }
}
