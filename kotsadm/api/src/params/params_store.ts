import pg from "pg";
import { Params } from "../server/params";

export class ParamsStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async getParam(name: string): Promise<string|null> {
    const q = `select value from kotsadm_params where key = $1`;
    const v = [name];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      return null
    }
    return result.rows[0].value;
  }

  async setParam(name: string, value: string) {
    const q = `
    insert into kotsadm_params (key, value) values ($1, $2)
    on conflict (key) do update set value = $2
    `;
    const v = [name, value];

    await this.pool.query(q, v);
  }

  async deleteParam(name: string) {
    const q = `delete from kotsadm_params where key = $1`;
    const v = [name];

    await this.pool.query(q, v);
  }
}
