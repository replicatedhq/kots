import { Pool, QueryResult } from "pg";
import { param } from "../params";

export async function getPostgresPool(): Promise<Pool> {
  const user = await param("POSTGRES_USER", "/shipcloud/postgres/rw_user");
  const host = await param("POSTGRES_HOST", "/shipcloud/postgres/host", true);
  const database = await param("POSTGRES_DATABASE", "/shipcloud/postgres/database");
  const port = await param("POSTGRES_PORT", "/shipcloud/postgres/port");
  const password = await param("POSTGRES_PASSWORD", "/shipcloud/postgres/rw_password", true);
  const uri = await param("POSTGRES_URI", "/shipcloud/postgres/uri", true);

  if (uri) {
    return new Pool({
      connectionString: uri,
    });
  } else {
    return new Pool({
      user,
      host,
      database,
      password,
      port: Number(port) || 5432,
    });
  }
}

export interface Querier {
  query(query: string, args?: any[]): Promise<QueryResult>;
}
