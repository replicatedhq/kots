import pg from "pg";
import { logger } from "../server/logger";
import { Params } from "../server/params";
import { Cluster } from "./";
import { ReplicatedError } from "../server/errors";
import randomstring from "randomstring";
import slugify from "slugify";
import _ from "lodash";

export class ClusterStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) {}

  async createOrUpdateHelmApplication(clusterId: string, helmApplication: any): Promise<void> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    const q = `select id from helm_chart where cluster_id = $1 and helm_name = $2`;
    const v = [
      clusterId,
      helmApplication.name,
    ]

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      return this.createHelmApplication(clusterId, helmApplication);
    } else {
      return this.updateHelmApplication(result.rows[0].id, helmApplication);
    }
  }

  async createHelmApplication(clusterId: string, helmApplication: any): Promise<void> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    let q = `insert into helm_chart (id, cluster_id, helm_name, namespace, version, first_deployed_at, last_deployed_at, is_deleted, chart_version, app_version)
      values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`;
    let v = [
      id,
      clusterId,
      helmApplication.name,
      helmApplication.namespace,
      helmApplication.version,
      helmApplication.firstDeployedAt,
      helmApplication.lastDeployedAt,
      helmApplication.isDeleted,
      helmApplication.chartVersion,
      helmApplication.appVersion,
    ]

    await this.pool.query(q, v);

    for (const source of helmApplication.sources) {
      q = `insert into helm_chart_source (helm_chart_id, source) values ($1, $2)`;
      v = [
        id,
        source,
      ];

      await this.pool.query(q, v);
    }
  }

  async updateHelmApplication(id: string, helmApplication: any): Promise<void> {
    let q = `update helm_chart set
      namespace = $1,
      version = $2,
      first_deployed_at = $3,
      last_deployed_at = $4,
      is_deleted = $5,
      chart_version = $6,
      app_version = $7
      where id = $8`;
    let v = [
      helmApplication.namespace,
      helmApplication.version,
      helmApplication.firstDeployedAt,
      helmApplication.lastDeployedAt,
      helmApplication.isDeleted,
      helmApplication.chartVersion,
      helmApplication.appVersion,
      id,
    ]

    await this.pool.query(q, v);

    q = `delete from helm_chart_source where helm_chart_id = $1`;
    v = [id];
    await this.pool.query(q, v);

    for (const source of helmApplication.sources) {
      q = `insert into helm_chart_source (helm_chart_id, source) values ($1, $2)`;
      v = [
        id,
        source,
      ];

      await this.pool.query(q, v);
    }
  }

  async listClustersForGitHubRepo(owner: string, repo: string): Promise<Cluster[]> {
    const q = `select cluster_id from cluster_github where owner = $1 and repo = $2`;
    const v = [
      owner,
      repo,
    ];

    const result = await this.pool.query(q, v);
    const clusterIds = result.rows.map(row => row.cluster_id);

    const clusters: Cluster[] = [];
    for (const clusterId of clusterIds) {
      clusters.push(await this.getGitOpsCluster(clusterId));
    }

    return clusters;
  }

  async getFromDeployToken(token: string): Promise<Cluster> {
    const q = `select id from cluster where token = $1`;
    const v = [
      token,
    ];

    const result = await this.pool.query(q, v);

    if (result.rows.length === 0) {
      throw new ReplicatedError("No cluster found");
    }
    
    return this.getCluster(result.rows[0].id);
  }

  async getGitOpsCluster(clusterId: string): Promise<Cluster> {
    const q = `select id, title, slug, created_at, updated_at, cluster_type, owner, repo, branch, installation_id
      from cluster
      inner join cluster_github on cluster_id = id
      where id = $1`;
    let v = [clusterId];

    const result = await this.pool.query(q, v);

    return this.mapCluster(result.rows[0]);
  }

  async maybeGetClusterWithTypeNameAndToken(clusterType: string, title: string, token: string): Promise<Cluster|void> {
    const q = `select id from cluster where cluster_type = $1 and title = $2 and token = $3`;
    const v = [
      clusterType,
      title,
      token,
    ];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      return;
    }

    return this.getShipOpsCluster(result.rows[0].id);
  }

  async getShipOpsCluster(clusterId: string): Promise<Cluster> {
    const q = `select id, title, slug, created_at, updated_at, token, cluster_type from cluster where id = $1`;
    const v = [clusterId];

    const result  = await this.pool.query(q, v);

    return this.mapCluster(result.rows[0]);
  }

  async listAllUsersClusters(): Promise<Cluster[]> {
      const q = `select id, cluster_type from cluster where is_all_users = true order by created_at, title`;
      const v = [];

      const result = await this.pool.query(q, v);
      const clusters: Cluster[] = [];
      for (const row of result.rows) {
        if (row.cluster_type === "gitops") {
          clusters.push(await this.getGitOpsCluster(row.id));
        } else {
          clusters.push(await this.getShipOpsCluster(row.id));
        }
      }

      return clusters;
  }

  async listClusters(userId: string): Promise<Cluster[]> {
    const q = `select id, cluster_type from cluster inner join user_cluster on cluster_id = id where user_cluster.user_id = $1 order by created_at, title`;
    const v = [userId];

    const result = await this.pool.query(q, v);
    const clusters: Cluster[] = [];
    for (const row of result.rows) {
      if (row.cluster_type === "gitops") {
        clusters.push(await this.getGitOpsCluster(row.id));
      } else {
        clusters.push(await this.getShipOpsCluster(row.id));
      }
    }

    return clusters;
  }

  async listClustersForKotsApp(appId: string): Promise<Cluster[]> {
    const q = `select cluster_id, c.id, c.cluster_type from app_downstream
      inner join cluster c on c.id = cluster_id
      where app_id = $1
      order by created_at, title`;
    const v = [appId];

    const result = await this.pool.query(q, v);
    const clusters: Cluster[] = [];
    for (const row of result.rows) {
      if (row.cluster_type === "gitops") {
        clusters.push(await this.getGitOpsCluster(row.id));
      } else {
        clusters.push(await this.getShipOpsCluster(row.id));
      }
    }

    return clusters;
  }

  async getCluster(id: string): Promise<Cluster> {
    const q = `select id, cluster_type from cluster where id = $1`;
    const v = [id];

    const result  = await this.pool.query(q, v);

    const clusterType = result.rows[0].cluster_type;

    switch (clusterType) {
      case "gitops":
        return this.getGitOpsCluster(result.rows[0].id);
      default:
        return this.getShipOpsCluster(result.rows[0].id);
    }
  }

  async getIdFromSlug(slug: string): Promise<string> {
    const q = "select id from cluster where slug = $1";
    const v = [slug];

    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No cluster with slug ${slug}`);
    }
    return result.rows[0].id;
  }

  async addUserToCluster(clusterId: string, userId: string): Promise<void> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      let q = `delete from user_cluster where user_id = $1 and cluster_id = $2`;
      let v: any[] = [
        userId,
        clusterId,
      ];
      await pg.query(q, v);

      q = `insert into user_cluster (user_id, cluster_id) values ($1, $2)`;
      v = [
        userId,
        clusterId,
      ];
      await pg.query(q, v);

      await pg.query("commit");
    } catch (err) {
      await pg.query("rollback");
      throw err;
    } finally {
      pg.release();
    }
  }

  async createNewShipCluster(userId: string|undefined, isAllUsers: boolean, title: string, token?: string): Promise<Cluster> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    let slugProposal = `${slugify(title, { lower: true })}`;

    let i = 0;
    let foundUniqueSlug = false;
    while (!foundUniqueSlug) {
      if (i > 0) {
        slugProposal = `${slugify(title, { lower: true })}-${i}`;
      }
      const qq = `select count(1) as count from cluster where slug = $1`;
      const vv = [
        slugProposal,
      ];

      const rr = await this.pool.query(qq, vv);
      if (parseInt(rr.rows[0].count) === 0) {
        foundUniqueSlug = true;
      }
      i++;
    }

    if (!token) {
      token = randomstring.generate({ capitalization: "lowercase" });
    }

    const pg = await this.pool.connect();
    await pg.query("begin");

    try {
      let q = `insert into cluster (id, title, slug, created_at, updated_at, cluster_type, is_all_users, token) values ($1, $2, $3, $4, $5, $6, $7, $8)`
      let v: any[] = [
        id,
        title,
        slugProposal,
        new Date(),
        null,
        "ship",
        isAllUsers,
        token,
      ];
      await pg.query(q, v);

      if (userId) {
        q = `insert into user_cluster (user_id, cluster_id) values ($1, $2)`;
        v = [
          userId,
          id,
        ];
        await pg.query(q, v);
      }

      await pg.query("commit");

      return this.getCluster(id);
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async createNewCluster(userId: string|undefined, isAllUsers: boolean, title: string, type: string, gitOwner?: string, gitRepo?: string, gitBranch?: string, gitInstallationId?: number): Promise<Cluster> {
    const id = randomstring.generate({ capitalization: "lowercase" });

    let slugProposal = `${slugify(title, { lower: true })}`;

    let i = 0;
    let foundUniqueSlug = false;
    while (!foundUniqueSlug) {
      if (i > 0) {
        slugProposal = `${slugify(title, { lower: true })}-${i}`;
      }
      const qq = `select count(1) as count from cluster where slug = $1`;
      const vv = [
        slugProposal,
      ];

      const rr = await this.pool.query(qq, vv);
      if (parseInt(rr.rows[0].count) === 0) {
        foundUniqueSlug = true;
      }
      i++;
    }

    const pg = await this.pool.connect();
    await pg.query("begin");

    try {
      let q = `insert into cluster (id, title, slug, created_at, updated_at, cluster_type, is_all_users) values ($1, $2, $3, $4, $5, $6, $7)`
      let v: any[] = [
        id,
        title,
        slugProposal,
        new Date(),
        null,
        type,
        isAllUsers,
      ];
      await pg.query(q, v);

      if (type === "ship") {
        const token = randomstring.generate({ capitalization: "lowercase" });
        q = `update cluster set token = $1 where id = $2`;
        v = [
          token,
          id,
        ];
        await pg.query(q, v);
      } else if (type === "gitops") {
        q = `insert into cluster_github (cluster_id, owner, repo, branch, installation_id) values ($1, $2, $3, $4, $5)`;
        v = [
          id,
          gitOwner!,
          gitRepo!,
          gitBranch!,
          gitInstallationId!,
        ];
        await pg.query(q, v);
      }

      if (userId) {
        q = `insert into user_cluster (user_id, cluster_id) values ($1, $2)`;
        v = [
          userId,
          id,
        ];
        await pg.query(q, v);
      }

      await pg.query("commit");

      return this.getCluster(id);
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async updateClusterGithubInstallationId(installationId: string, owner: string): Promise<void> {
    const pg = await this.pool.connect();
    const q = `
UPDATE cluster_github
SET installation_id = $1, is_deleted = FALSE, is_404 = FALSE
WHERE cluster_github.owner = $2 AND cluster_github.is_deleted = TRUE`;
    const v = [installationId, owner];
    const res = await pg.query(q, v);
    if (res.rowCount > 0) {
      logger.info({msg: `Updated ${res.rowCount} row(s) with new installation ${installationId}`});
    }
  }

  async updateClusterGithubInstallationRepoAdded(installationId: number, owner: string, repo: string): Promise<void> {
    const pg = await this.pool.connect();
    const q = `
UPDATE cluster_github
SET is_404 = FALSE
WHERE installation_id = $1 AND owner = $2 AND repo = $3 AND is_404 = TRUE`;
    const v = [installationId, owner, repo];
    const res = await pg.query(q, v);
    if (res.rowCount > 0) {
      logger.info({msg: `Updated ${res.rowCount} row(s) with is_404 false for repo ${owner}/${repo} and installation ${installationId}`});
    }
  }

  async updateClusterGithubInstallationsAsDeleted(installationId: string): Promise<void> {
    const pg = await this.pool.connect();
    const q = `UPDATE cluster_github SET is_deleted = TRUE WHERE installation_id = $1 AND (is_deleted = FALSE OR is_deleted is NULL)`;
    const v = [installationId];
    const res = await pg.query(q, v);
    if (res.rowCount > 0) {
      logger.info({msg: `Marked ${res.rowCount} row(s) for installation ${installationId} as is_deleted`});
    }
  }

  async getOperatorInstallationManifests(clusterId: string): Promise<string> {
    const cluster = await this.getShipOpsCluster(clusterId);

    const manifests = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kotsadm-operator
spec:
  selector:
    matchLabels:
      app: kotsadm-operator
  template:
    metadata:
      labels:
        app: kotsadm-operator
    spec:
      containers:
      - env:
        - name: KOTSADM_API_ENDPOINT
          value: ${this.params.shipApiEndpoint}
        - name: KOTSADM_TOKEN
          value: ${cluster.shipOpsRef!.token}
        image: kotsadm-operator
        imagePullPolicy: Always
        name: kotsadm-operator
        resources:
          limits:
            cpu: 200m
            memory: 1000Mi
          requests:
            cpu: 100m
            memory: 500Mi
      restartPolicy: Always

`;

    return manifests;
  }

  async updateCluster(userId: string, clusterId: string, clusterName: string): Promise<boolean> {
    const slugProposal = `${slugify(clusterName, { lower: true })}`;
    const clusters = await this.listClusters(userId);
    const existingSlugs = clusters.map(cluster => cluster.slug);
    let finalSlug = slugProposal;

    if (_.includes(existingSlugs, slugProposal)) {
      const maxNumber =
        _(existingSlugs)
          .map(slug => {
            const result = slug!.replace(slugProposal, "").replace("-", "");

            return result ? parseInt(result, 10) : 0;
          })
          .max() || 0;

      finalSlug = `${slugProposal}-${maxNumber + 1}`;
    }

    const q = `update cluster set title = $1, slug = $2, updated_at = $3 where id = $4`;
    const v = [clusterName, finalSlug, new Date(), clusterId];
    await this.pool.query(q, v);

    return true;
  }

  async getApplicationCount(clusterId: string): Promise<number> {
    let q = `select count(1) as count from watch_cluster where cluster_id = $1`;
    let v = [clusterId];
    let result = await this.pool.query(q, v);
    const wcCount = parseInt(result.rows[0].count);

    q = `select count(1) as count from app_downstream where cluster_id = $1`;
    v = [clusterId];
    result = await this.pool.query(q, v);
    const adCount = parseInt(result.rows[0].count);

    return wcCount + adCount;
  }

  async deleteCluster( userId: string, clusterId: string): Promise<boolean> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      const applicationsCount = await this.getApplicationCount(clusterId);
      if (applicationsCount > 0) {
        throw new ReplicatedError("This cluster has applications deployed to it so it cannot be deleted.");
      }

      try {
        const cluster = await this.getCluster(clusterId);

        let q = `delete from cluster where id = $1`;
        let v = [clusterId];
        await pg.query(q, v);

        q = `delete from user_cluster where user_id = $1 and cluster_id = $2`;
        v = [userId, clusterId];
        await pg.query(q, v);

        await pg.query("commit");
      } catch (err) {
        await pg.query("rollback");
        throw err;
      }

      return true;
    } finally {
      pg.release();
    }
  }

  private mapCluster(row: any): Cluster {
    let shipOpsRef: any = null
    if (row.token) {
      shipOpsRef = {
        token: row.token
      }
    }

    const c = new Cluster();
    c.id = row.id;
    c.title = row.title;
    c.slug = row.slug;
    c.createdOn = row.created_at;
    c.lastUpdated = row.updated_at;
    c.shipOpsRef = shipOpsRef;
    return c;
  }
}
