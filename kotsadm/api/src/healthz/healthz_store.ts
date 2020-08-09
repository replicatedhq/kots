import pg from "pg";
import { Params } from "../server/params";
import { bucketExists } from "../util/s3";

export interface DatabaseInfo {
  database: {
    connected: boolean,
  },
  storage: {
    available: boolean,
  },
}

export class HealthzStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
  ) {}

  async getHealthz(): Promise<DatabaseInfo> {
    const query = `select count(1)`;
    await this.pool.query(query);

    let storageReady = false;

    if (this.params.storageBaseURI.startsWith("docker://")) {
      storageReady = true;  // TODO
    } else {
      storageReady = await bucketExists(this.params, this.params.shipOutputBucket);
    }

    return {
      database: {
        connected: true,
      },
      storage: {
        available: storageReady,
      }
    };
  }
}
