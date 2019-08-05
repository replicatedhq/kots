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
    return this.getCluster(result.rows[0].id);
  }

  async getGitOpsCluster(clusterId: string, watchId?: string): Promise<Cluster> {
    const fields = ["id", "title", "slug", "created_at", "updated_at", "cluster_type", "owner", "repo", "branch", "installation_id"]
    if (watchId) {
      fields.push("wc.github_path");
    }

    const q = `select ${fields} from cluster
      inner join
        cluster_github on cluster_id = id
      ${watchId ? " left outer join watch_cluster as wc on wc.cluster_id = id where wc.cluster_id = $1 and watch_id = $2" : " where id = $1"}`;

    let v = [clusterId];
    if (watchId) {
      v.push(watchId);
    }

    const result = await this.pool.query(q, v);

    return this.mapCluster(result.rows[0]);
  }

  async getLocalShipOpsCluster(): Promise<Cluster|void> {
    const q = `select id from cluster where cluster_type = $1 and title = $2`;
    const v = [
      "ship",
      "This Cluster",
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

  async getForWatch(watchId: string): Promise<Cluster | void> {
    const q = `select cluster_id, cluster_type from watch_cluster inner join cluster on cluster_id = id where watch_id = $1`;
    const v = [watchId];

    const result  = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      return;
    }
    let cluster: Cluster;
    if (result.rows[0].cluster_type === "gitops") {
      cluster = await this.getGitOpsCluster(result.rows[0].cluster_id, watchId);
    } else {
      cluster = await this.getShipOpsCluster(result.rows[0].cluster_id);
    }

    return cluster;
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
    const q = `UPDATE cluster_github SET is_deleted = true WHERE installation_id = $1 AND (is_deleted = false OR is_deleted is NULL)`;
    const v = [installationId];
    const res = await pg.query(q, v);
    if (res.rowCount > 0) {
      logger.info({msg: `Marked ${res.rowCount} row(s) for installation ${installationId} as is_deleted`});
    }
  }

  async getShipInstallationManifests(clusterId: string): Promise<string> {
    const cluster = await this.getShipOpsCluster(clusterId);

    const manifests = `
apiVersion: v1
kind: List
items:
  - apiVersion: v1
    kind: Namespace
    metadata:
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
      name: ship-cd-system
  - apiVersion: apiextensions.k8s.io/v1beta1
    kind: CustomResourceDefinition
    metadata:
      creationTimestamp: null
      labels:
        controller-tools.k8s.io: "1.0"
      name: clusters.clusters.replicated.com
    spec:
      group: clusters.replicated.com
      names:
        kind: Cluster
        plural: clusters
      scope: Namespaced
      validation:
        openAPIV3Schema:
          properties:
            apiVersion:
              description: 'APIVersion defines the versioned schema of this representation
                of an object. Servers should convert recognized schemas to the latest
                internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
              type: string
            kind:
              description: 'Kind is a string value representing the REST resource this
                object represents. Servers may infer this from the endpoint the client
                submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
              type: string
            metadata:
              type: object
            spec:
              properties:
                shipApiServer:
                  type: string
                token:
                  type: string
              required:
              - shipApiServer
              - token
              type: object
            status:
              type: object
      version: v1alpha1
    status:
      acceptedNames:
        kind: ""
        plural: ""
      conditions: []
      storedVersions: []
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      creationTimestamp: null
      name: ship-cd-manager-role
    rules:
    - apiGroups: ['*']
      resources: ['*']
      verbs: ['*']
    - nonResourceURLs: ['*']
      verbs: ['*']
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: ship-cd-proxy-role
    rules:
    - apiGroups:
      - authentication.k8s.io
      resources:
      - tokenreviews
      verbs:
      - create
    - apiGroups:
      - authorization.k8s.io
      resources:
      - subjectaccessreviews
      verbs:
      - create
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      creationTimestamp: null
      name: ship-cd-manager-rolebinding
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: ship-cd-manager-role
    subjects:
    - kind: ServiceAccount
      name: default
      namespace: ship-cd-system
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: ship-cd-proxy-rolebinding
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: ship-cd-proxy-role
    subjects:
    - kind: ServiceAccount
      name: default
      namespace: ship-cd-system
  - apiVersion: v1
    kind: Secret
    metadata:
      name: ship-cd-webhook-server-secret
      namespace: ship-cd-system
  - apiVersion: v1
    kind: Service
    metadata:
      annotations:
        prometheus.io/port: "8443"
        prometheus.io/scheme: https
        prometheus.io/scrape: "true"
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
      name: ship-cd-controller-manager-metrics-service
      namespace: ship-cd-system
    spec:
      ports:
      - name: https
        port: 8443
        targetPort: https
      selector:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
  - apiVersion: v1
    kind: Service
    metadata:
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
      name: ship-cd-controller-manager-service
      namespace: ship-cd-system
    spec:
      ports:
      - port: 443
      selector:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
  - apiVersion: apps/v1
    kind: StatefulSet
    metadata:
      labels:
        control-plane: controller-manager
        controller-tools.k8s.io: "1.0"
      name: ship-cd-controller-manager
      namespace: ship-cd-system
    spec:
      selector:
        matchLabels:
          control-plane: controller-manager
          controller-tools.k8s.io: "1.0"
      serviceName: ship-cd-controller-manager-service
      template:
        metadata:
          labels:
            control-plane: controller-manager
            controller-tools.k8s.io: "1.0"
        spec:
          containers:
          - args:
            - --secure-listen-address=0.0.0.0:8443
            - --upstream=http://127.0.0.1:8080/
            - --logtostderr=true
            - --v=10
            image: gcr.io/kubebuilder/kube-rbac-proxy:v0.4.0
            name: kube-rbac-proxy
            ports:
            - containerPort: 8443
              name: https
          - args:
            - --metrics-addr=127.0.0.1:8080
            command:
            - /manager
            env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: SECRET_NAME
              value: ship-cd-webhook-server-secret
            image: replicated/ship-cd:latest
            imagePullPolicy: Always
            name: manager
            ports:
            - containerPort: 9876
              name: webhook-server
              protocol: TCP
            resources:
              limits:
                cpu: 100m
                memory: 500Mi
              requests:
                cpu: 100m
                memory: 500Mi
            volumeMounts:
            - mountPath: /tmp/cert
              name: cert
              readOnly: true
          terminationGracePeriodSeconds: 10
          volumes:
          - name: cert
            secret:
              defaultMode: 420
              secretName: ship-cd-webhook-server-secret
  - apiVersion: clusters.replicated.com/v1alpha1
    kind: Cluster
    metadata:
      labels:
        controller-tools.k8s.io: "1.0"
      name: ${slugify(cluster.title!, { lower: true })}
    spec:
      shipApiServer: ${this.params.shipApiEndpoint}
      token: ${cluster.shipOpsRef!.token}
`;

    return manifests;
  }

  async updateCluster(userId: string, clusterId: string, clusterName: string, gitOpsRef: any): Promise<boolean> {
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

    if (gitOpsRef) {
      const q = `update cluster_github set owner = $1, repo = $2, branch = $3 where cluster_id = $4`;
      const v = [gitOpsRef.owner, gitOpsRef.repo, gitOpsRef.branch, clusterId];
      await this.pool.query(q, v);
    }

    return true;
  }

  async getApplicationCount(clusterId: string): Promise<number> {
    const q = `select count(1) as count from watch_cluster where cluster_id = $1`;
    const v = [clusterId];
    const { rows }: { rows: any[] }  = await this.pool.query(q, v);

    return rows[0].count;
  }

  async deleteCluster( userId: string, clusterId: string): Promise<boolean> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      const q = `select count(1) as count from watch_cluster where cluster_id = $1`;
      const v = [clusterId];
      const { rows }: { rows: any[] } = await pg.query(q, v);

      if (rows[0].count > 0) {
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

        if (cluster.gitOpsRef) {
          const q = `delete from cluster_github where cluster_id = $1`;
          const v = [clusterId];
          await pg.query(q, v);
        }

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
    let gitOpsRef, shipOpsRef: any = null
    if (row.token) {
      shipOpsRef = {
        token: row.token
      }
    }
    if (row.cluster_type === "gitops") {
      gitOpsRef = {
        owner: row.owner,
        repo: row.repo,
        branch: row.branch,
        path: row.github_path || "",
        installationId: row.installation_id,
      }
    }
    return {
      id: row.id,
      title: row.title,
      slug: row.slug,
      createdOn: row.created_at,
      lastUpdated: row.updated_at,
      gitOpsRef,
      shipOpsRef
    };
  }
}
