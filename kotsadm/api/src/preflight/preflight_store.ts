import pg from "pg";

import { Params } from "../server/params";

interface PreflightParams {
  url: string
  ignorePermissions: boolean
}

export class PreflightStore {
  constructor(
    private readonly pool: pg.Pool
  ) {}

  async getPendingPreflightParams(inCluster: boolean): Promise<PreflightParams[]> {
    const params = await Params.getParams();
    const q =
      `SELECT
        app_downstream_version.sequence as sequence,
        app_downstream_version.preflight_ignore_permissions,
        app.slug as app_slug,
        cluster.slug as cluster_slug
      FROM app_downstream_version
        INNER JOIN app ON app_downstream_version.app_id = app.id
        INNER JOIN cluster ON app_downstream_version.cluster_id = cluster.id
      WHERE app_downstream_version.status = 'pending_preflight'`;

    const result = await this.pool.query(q);

    const preflightParams: PreflightParams[] = [];
    for (const row of result.rows) {
      const {
        app_slug: appSlug,
        cluster_slug: clusterSlug,
        sequence,
        preflight_ignore_permissions: ignorePermissions,
      } = row;

      let url: string;
      if (inCluster) {
        url = `${params.shipApiEndpoint}/api/v1/preflight/${appSlug}/sequence/${sequence}?incluster=true`;
      } else {
        url = `${params.apiAdvertiseEndpoint}/api/v1/preflight/${appSlug}/sequence/${sequence}`;
      }
  
      const param: PreflightParams = {
        url: url,
        ignorePermissions: ignorePermissions,
      };
      preflightParams.push(param);
    }

    return preflightParams;
  }

}
