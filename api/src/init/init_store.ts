import * as randomstring from "randomstring";
import rp from "request-promise";
import { InitSession } from "./";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import * as pg from "pg";
import { ReplicatedError } from "../server/errors";

export class InitStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params
  ) {
  }

  async createInitSession(userId: string, upstreamUri: string, clusterId: any, githubPath: any, parentWatchId?: string, parentSequence?: number, requestedUpstreamUri?: string): Promise<InitSession> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `insert into ship_init (id, user_id, upstream_uri, created_at, cluster_id, github_path, requested_upstream_uri, parent_watch_id, parent_sequence) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`;
    const v = [
      id,
      userId,
      upstreamUri,
      new Date(),
      clusterId,
      githubPath,
      requestedUpstreamUri,
      parentWatchId,
      parentSequence,
    ];

    await this.pool.query(q, v);

    return this.getSession(id);
  }

  async deployInitSession(id: string, pendingInitId?: string): Promise<InitSession> {
    const initSession = await this.getSession(id);

    if (this.params.skipDeployToWorker) {
      logger.info({msg: "skipping deploy to worker"});
      return initSession;
    }

    const body: any = {};
    body.id = initSession.id;
    body.upstreamUri = initSession.upstreamURI;
    if (pendingInitId) {
      body.pendingInitId = pendingInitId;
    }
    const options = {
      method: "POST",
      uri: `${this.params.shipInitBaseURL}/v1/init`,
      body,
      json: true,
    };

    const parsedBody = await rp(options);
    logger.debug({ msg: "response from deploy init", parsedBody });
    return initSession;
  }

  async getSession(id: string): Promise<InitSession> {
    const q = `select id, upstream_uri, created_at, finished_at, result from ship_init where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);

    if (result.rowCount === 0) {
      throw new ReplicatedError(`Init session ${id} not found`);
    }

    return {
      id: result.rows[0].id,
      upstreamURI: result.rows[0].upstream_uri,
      createdOn: result.rows[0].created_at,
      finishedOn: result.rows[0].finished_at,
      result: result.rows[0].result,
    };
  }
}
