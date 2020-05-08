import { Pool, QueryResult } from "pg";
import { Params } from "../../server/params";

export async function getPostgresPool(): Promise<Pool> {
  const uri = (await Params.getParams()).postgresUri;
  return new Pool({
    connectionString: uri,
  });
}

export interface Querier {
  query(query: string, args?: any[]): Promise<QueryResult>;
}
