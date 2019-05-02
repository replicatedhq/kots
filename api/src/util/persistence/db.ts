import { Pool, PoolClient, QueryConfig, QueryResult } from "pg";
import { logger } from "../../server/logger";
import { param } from "../params";

export async function getPostgresPool(): Promise<Pool> {
  const user = await param("POSTGRES_USER", "/shipcloud/postgres/rw_user");
  const host = await param("POSTGRES_HOST", "/shipcloud/postgres/host", true);
  const database = await param("POSTGRES_DATABASE", "/shipcloud/postgres/database");
  const port = await param("POSTGRES_PORT", "/shipcloud/postgres/port");
  const password = await param("POSTGRES_PASSWORD", "/shipcloud/postgres/rw_password", true);
  const uri = await param("POSTGRES_URI", "/shipcloud/postgres/uri", true);

  if (uri) {
    logger.info(`Connecting to postgres with connection string set in uri`);
    return new Pool({
      connectionString: uri,
    });
  } else {
    logger.info(`Connecting to postgres with connection string: server=${host};uid=${user};pwd=*******;database=${database}`);
    return new Pool({
      user,
      host,
      database,
      password,
      port: Number(port) || 5432,
    });
  }
}

export class PostgresWrapper {
  client?: PoolClient;

  constructor(readonly pool: Pool) {}

  withClient(client: PoolClient): PostgresWrapper {
    const newWrapper = new PostgresWrapper(this.pool);
    newWrapper.client = client;

    return newWrapper;
  }

  async query(q: string | QueryConfig, v?: any[]): Promise<QueryResult> {
    if (!q) {
      const msg = `falsey ${q} passed as query`;
      logger.info({msg, q});
      throw new Error(msg)
    }
    if (this.client) {
      return this.client.query(q, v);
    }

    return this.pool.query(q, v);
  }
}

export async function transaction<TResult>(wrapper: PostgresWrapper, query: (wrapperWithClient: PostgresWrapper) => Promise<TResult>) {
  const clientToUse = await wrapper.pool.connect();
  const newWrapper = wrapper.withClient(clientToUse);

  try {
    await clientToUse.query("BEGIN");
    const result: TResult = await query(newWrapper);
    await clientToUse.query("COMMIT");

    return result;
  } catch (e) {
    await clientToUse.query("ROLLBACK");
    throw e;
  } finally {
    clientToUse.release();
  }
}

// Connectable describes an object that can get a poolconnection,
// and create a copy of itself that uses a specified connection
// instead of its underlying data source
export interface Connectable<TStore> {
  wrapper: PostgresWrapper;
  withWrapper(wrapper: PostgresWrapper): TStore;
}

export type StoreTransaction<T, R extends Connectable<R>> = (store: R) => Promise<T>;

// Store transaction takes a store that implements Connectable
// and runs the given StoreTransaction, managing rollbacks and connection closing
export async function storeTransaction<TResult, TStore extends Connectable<TStore>>(store: TStore, txn: StoreTransaction<TResult, TStore>): Promise<TResult> {
  const clientToUse = await store.wrapper.pool.connect();
  const newWrapper = store.wrapper.withClient(clientToUse);
  const newStore = store.withWrapper(newWrapper);

  try {
    await clientToUse.query("BEGIN");
    const result: TResult = await txn(newStore);
    await clientToUse.query("COMMIT");

    return result;
  } catch (e) {
    await clientToUse.query("ROLLBACK");
    throw e;
  } finally {
    clientToUse.release();
  }
}
