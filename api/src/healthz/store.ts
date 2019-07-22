import pg from "pg";

interface DatabaseInfo {
  connected: boolean;
}

export class HealthzStore {
  constructor(private readonly pool: pg.Pool) {}

  async getHealthz(): Promise<DatabaseInfo> {
    const query = `select count(1)`;
    await this.pool.query(query);

    return {
      connected: true,
    };
  }
}
