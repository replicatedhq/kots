import { Pool } from "pg";
import { Controller, Get } from "ts-express-decorators";

interface DatabaseInfo {
  version: string;
  dirty: boolean;
  connected: boolean;
}

@Controller("/healthz")
export class HealthzAPI {
  constructor(private readonly pool: Pool) {}

  @Get("")
  private async getDatabaseInfo(): Promise<DatabaseInfo> {
    const query = `SELECT "version", dirty
                   FROM schema_migrations
                   LIMIT 1`;
    const res = await this.pool.query(query);

    let version = "unknown";
    let dirty = false;

    const rows = res.rows;
    if (rows && rows.length > 0) {
      version = rows[0].version;
      dirty = rows[0].dirty;
    }

    return {
      version,
      dirty,
      connected: true,
    };
  }
}
