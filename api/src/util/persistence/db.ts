import { Pool, QueryResult } from "pg";
import { param } from "../params";

export async function getPostgresPool(): Promise<Pool> {
  const uri = await param("POSTGRES_URI", "/shipcloud/postgres/uri", true);
  return new Pool({
    connectionString: uri,
  });
}

export interface Querier {
  query(query: string, args?: any[]): Promise<QueryResult>;
}
