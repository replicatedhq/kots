import pg from "pg";
import { Params } from "../server/params";
import { KotsApp, KotsVersion, KotsAppRegistryDetails, KotsDownstreamOutput, ConfigData } from "./";
import { ReplicatedError } from "../server/errors";
import randomstring from "randomstring";
import slugify from "slugify";
import * as k8s from "@kubernetes/client-node";
import { kotsEncryptString, kotsDecryptString } from "./kots_ffi"
import _ from "lodash";
import yaml from "js-yaml";
import { base64Decode, getPreflightResultState, base64Encode } from '../util/utilities';
import { ApplicationSpec } from "./kots_app_spec";
import { InstallationSpec } from "./kots_installation_spec";
import { logger } from "../server/logger";
import { KotsAppVersion } from "./kots_app";

export enum UndeployStatus {
  InProcess = "in_process",
  Completed = "completed",
  Failed = "failed",
}

interface UpdateCursor {
  cursor: string;
  channelName: string;
}

export class KotsAppStore {
  constructor(private readonly pool: pg.Pool, private readonly params: Params) { }

  async createGitOpsRepo(provider: string, uri: string, hostname: string, privateKey: string, publicKey: string): Promise<void> {
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

      const namespace = process.env["POD_NAMESPACE"]!;
      const secretName = "kotsadm-gitops";
      let secretExists = false;
      let data: { [key: string]: string } = {};

      try {
        // read secret data (if exists)
        const secret = await k8sApi.readNamespacedSecret(secretName, namespace);
        data = secret.body.data || {};
        secretExists = true;
      } catch (err) {
        // secret does not exist yet
      }

      const keys = Object.keys(data); // key example: "provider.0.type"

      let index = -1, repoExists = false;
      for (const key of keys) {
        const value = base64Decode(data[key]);
        if (value === uri) {
          index = parseInt(key.charAt(9));
          repoExists = true;
          break;
        }
      }

      if (index === -1) {
        const indices = _.map(keys, key => parseInt(key.charAt(9)));
        if (indices.length) {
          index = Math.max(...indices) + 1;
        } else {
          index = 0;
        }
      }

      data[`provider.${index}.type`] = base64Encode(provider);
      data[`provider.${index}.repoUri`] = base64Encode(uri);

      if (!repoExists) {
        data[`provider.${index}.publicKey`] = base64Encode(publicKey);
        data[`provider.${index}.privateKey`] = base64Encode(privateKey);
      }

      const hostnameKey = `provider.${index}.hostname`;
      if (hostnameKey in data) {
        delete data[hostnameKey];
      }

      if (hostname) {
        data[hostnameKey] = base64Encode(hostname);
      }

      const secretObj: k8s.V1Secret = {
        apiVersion: "v1",
        kind: "Secret",
        metadata: {
          name: secretName,
        },
        data: data
      }

      if (!secretExists) {
        await k8sApi.createNamespacedSecret(namespace, secretObj);
      } else {
        await k8sApi.replaceNamespacedSecret(secretName, namespace, secretObj);
      }
    } catch (err) {
      const msg = _.get(err, "response.body.message");
      throw new ReplicatedError(`Failed to create gitops secret ${msg || String(err)}`);
    }
  }

  async setGitOpsError(appId: string, clusterId: string, err: any): Promise<void> {
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

      const namespace = process.env["POD_NAMESPACE"]!;
      const configMapName = "kotsadm-gitops";
      const configmap = await k8sApi.readNamespacedConfigMap(configMapName, namespace);
      const configMapData = configmap.body.data!;

      const downstreamData = JSON.parse(base64Decode(configMapData[`${appId}-${clusterId}`]));
      downstreamData.lastError = err;

      configMapData[`${appId}-${clusterId}`] = base64Encode(JSON.stringify(downstreamData));

      const configMapObj: k8s.V1Secret = {
        apiVersion: "v1",
        kind: "ConfigMap",
        metadata: {
          name: configMapName,
        },
        data: configMapData
      }

      await k8sApi.replaceNamespacedConfigMap(configMapName, namespace, configMapObj);
    } catch (err) {
      logger.error(err);
      throw new ReplicatedError("Failed to set gitops error");
    }
  }

  async resetGitOpsData(): Promise<void> {
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

      const namespace = process.env["POD_NAMESPACE"]!;

      try {
        const secretName = "kotsadm-gitops";
        await k8sApi.deleteNamespacedSecret(secretName, namespace);
      } catch (err) {
        // secret does not exist
      }

      try {
        const configMapName = "kotsadm-gitops";
        await k8sApi.deleteNamespacedConfigMap(configMapName, namespace);
      } catch (err) {
        // config map does not exist
      }
    } catch (err) {
      logger.error(err);
      throw new ReplicatedError("Failed to reset gitops data");
    }
  }

  async disableDownstreamGitOps(appId: string, clusterId: string): Promise<void> {
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

      const namespace = process.env["POD_NAMESPACE"]!;
      const configMapName = "kotsadm-gitops";
      const configmap = await k8sApi.readNamespacedConfigMap(configMapName, namespace);
      const configMapData = configmap.body.data!;

      delete configMapData[`${appId}-${clusterId}`];

      const configMapObj: k8s.V1Secret = {
        apiVersion: "v1",
        kind: "ConfigMap",
        metadata: {
          name: configMapName,
        },
        data: configMapData
      }

      await k8sApi.replaceNamespacedConfigMap(configMapName, namespace, configMapObj);
    } catch (err) {
      logger.error(err);
      throw new ReplicatedError(`Failed to disable gitops for app with id ${appId}`);
    }
  }

  async getGitOpsCreds(appId: string, clusterId: string): Promise<any> {
    const { repoUri, provider, privateKey, publicKey } = await this.getGitopsInfo(appId, clusterId);

    let cloneUri = repoUri;  // this is unlikely to work because we only support ssh auth later.  hmmm
    const uriParts = repoUri.split("/");

    switch (provider) {
      case "github":
        cloneUri = `git@github.com:${uriParts[3]}/${uriParts[4]}.git`;
        break;
      case "github_enterprise":
        cloneUri = `git@${uriParts[2]}:${uriParts[3]}/${uriParts[4]}.git`;
        break;
      case "gitlab":
        cloneUri = `git@gitlab.com:${uriParts[3]}/${uriParts[4]}.git`;
        break;
      case "bitbucket":
        cloneUri = `git@bitbucket.org:${uriParts[3]}/${uriParts[4]}.git`;
        break;
    }

    return {
      uri: repoUri,
      pubKey: publicKey,
      privKey: privateKey,
      provider: provider,
      cloneUri,
    };
  }

  async setDownstreamGitOps(appId: string, clusterId: string, repoUri: string, branch: string, path: string, format: string, action: string): Promise<any> {
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

      const namespace = process.env["POD_NAMESPACE"]!;
      const configMapName = "kotsadm-gitops";
      let data: { [key: string]: string } = {};
      let configMapExists = false;

      try {
        // read config map data (if exists)
        const configmap = await k8sApi.readNamespacedConfigMap(configMapName, namespace);
        data = configmap.body.data || {};
        configMapExists = true;
      } catch (err) {
        // configmap does not exist yet
      }

      const key = `${appId}-${clusterId}`;

      let lastError = {};
      if (key in data) {
        const parsedData = JSON.parse(base64Decode(data[key]));
        const oldUri = parsedData.repoUri;
        const oldBranch = parsedData.branch;
        if (oldBranch !== branch || oldUri !== repoUri) {
          lastError = {}; // reset last error
        } else {
          lastError = { lastError: parsedData.lastError }; // keep last error
        }
      }

      data[key] = base64Encode(JSON.stringify({
        repoUri: repoUri,
        branch: branch,
        path: path,
        format: format,
        action: action,
        ...lastError
      }));

      const configMapObj: k8s.V1Secret = {
        apiVersion: "v1",
        kind: "ConfigMap",
        metadata: {
          name: configMapName,
        },
        data: data
      }

      if (!configMapExists) {
        await k8sApi.createNamespacedConfigMap(namespace, configMapObj);
      } else {
        await k8sApi.replaceNamespacedConfigMap(configMapName, namespace, configMapObj);
      }
    } catch (err) {
      const msg = _.get(err, "response.body.message");
      throw new ReplicatedError(`Failed to create gitops configmap ${msg || String(err)}`);
    }
  }

  async listClusterIDsForApp(id: string): Promise<string[]> {
    const q = `select cluster_id from app_downstream where app_id = $1`;
    const v = [
      id,
    ];

    const result = await this.pool.query(q, v);
    const clusterIds: string[] = [];
    for (const row of result.rows) {
      clusterIds.push(row.cluster_id);
    }

    return clusterIds;
  }

  async listAppsForCluster(clusterId: string): Promise<KotsApp[]> {
    const q = `select app_id from app_downstream where cluster_id = $1`;
    const v = [
      clusterId,
    ];

    const result = await this.pool.query(q, v);
    const apps: KotsApp[] = [];
    for (const row of result.rows) {
      apps.push(await this.getApp(row.app_id));
    }

    return apps;
  }

  async listApps(): Promise<KotsApp[]> {
    const q = `select id from app`;
    const v = [];

    const result = await this.pool.query(q, v);
    const apps: KotsApp[] = [];
    for (const row of result.rows) {
      apps.push(await this.getApp(row.id));
    }

    return apps;
  }

  async listAppSlugs(): Promise<string[]> {
    const q = `select slug from app`;
    const v = [];

    const result = await this.pool.query(q,v);
    return _.map(result.rows, ({ slug }) => slug);
  }

  async updateDownstreamsStatus(appId: string, sequence: number, status: string, statusInfo: string): Promise<void> {
    const q = `
      update app_downstream_version
      set status = $3, status_info = $4
      where app_id = $1 and sequence = $2
    `;
    const v = [
      appId,
      sequence,
      status,
      statusInfo,
    ];
    await this.pool.query(q, v);
  }

  async getDownstreamOutput(appId: string, clusterId: string, sequence: number): Promise<KotsDownstreamOutput> {
    const q = `
      select adv.status, adv.status_info, ado.dryrun_stdout, ado.dryrun_stderr, ado.apply_stdout, ado.apply_stderr
      from app_downstream_version adv LEFT JOIN app_downstream_output ado
        ON adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
      where adv.app_id = $1 and adv.cluster_id = $2 and adv.sequence = $3
    `;
    const v = [
      appId,
      clusterId,
      sequence,
    ];
    const result = await this.pool.query(q, v);

    if (result.rows.length === 0) {
      return {
        dryrunStdout: "",
        dryrunStderr: "",
        applyStdout: "",
        applyStderr: "",
        renderError: ""
      };
    };

    const row = result.rows[0];

    let renderError: string | null;
    if (row.status === "failed") {
      renderError = row.status_info;
    } else {
      renderError = null;
    }

    return {
      dryrunStdout: base64Decode(row.dryrun_stdout),
      dryrunStderr: base64Decode(row.dryrun_stderr),
      applyStdout: base64Decode(row.apply_stdout),
      applyStderr: base64Decode(row.apply_stderr),
      renderError: renderError,
    };
  }

  async createDownstream(appId: string, downstreamName: string, clusterId: string): Promise<void> {
    const q = `insert into app_downstream (app_id, downstream_name, cluster_id)
    values ($1, $2, $3)
    ON CONFLICT(app_id, cluster_id) DO UPDATE SET
    downstream_name = EXCLUDED.downstream_name`;
    const v = [
      appId,
      downstreamName,
      clusterId,
    ];

    await this.pool.query(q, v);
  }

  async createMidstreamVersion(
    id: string,
    sequence: number,
    versionLabel: string,
    releaseNotes: string,
    updateCursor: string,
    channelName: string,
    encryptionKey: string,
    supportBundleSpec: any,
    analyzersSpec: any,
    preflightSpec: any,
    appSpec: any,
    kotsAppSpec: string | null,
    kotsInstallationSpec: string | null,
    kotsAppLicense: any,
    configSpec: any,
    configValues: any,
    appTitle: string | null,
    appIcon: string | null,
    backupSpec: any,
  ): Promise<void> {
    const q = `insert into app_version (app_id, sequence, created_at, version_label, release_notes, update_cursor, channel_name, encryption_key,
        supportbundle_spec, analyzer_spec, preflight_spec, app_spec, kots_app_spec, kots_installation_spec, kots_license, config_spec, config_values, backup_spec)
      values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
      ON CONFLICT(app_id, sequence) DO UPDATE SET
      created_at = EXCLUDED.created_at,
      version_label = EXCLUDED.version_label,
      release_notes = EXCLUDED.release_notes,
      update_cursor = EXCLUDED.update_cursor,
      channel_name = EXCLUDED.channel_name,
      encryption_key = EXCLUDED.encryption_key,
      supportbundle_spec = EXCLUDED.supportbundle_spec,
      analyzer_spec = EXCLUDED.analyzer_spec,
      preflight_spec = EXCLUDED.preflight_spec,
      app_spec = EXCLUDED.app_spec,
      kots_app_spec = EXCLUDED.kots_app_spec,
      kots_installation_spec = EXCLUDED.kots_installation_spec,
      kots_license = EXCLUDED.kots_license,
      config_spec = EXCLUDED.config_spec,
      config_values = EXCLUDED.config_values,
      backup_spec = EXCLUDED.backup_spec
    `;
    const v = [
      id,
      sequence,
      new Date(),
      versionLabel,
      releaseNotes,
      updateCursor,
      channelName,
      encryptionKey,
      supportBundleSpec,
      analyzersSpec,
      preflightSpec,
      appSpec,
      kotsAppSpec,
      kotsInstallationSpec,
      kotsAppLicense,
      configSpec,
      configValues,
      backupSpec,
    ];

    await this.pool.query(q, v);

    let name;
    if (!appTitle) {
      const qqq = `select slug from app where id = $1`;
      const vvv = [id];

      const result = await this.pool.query(qqq, vvv);
      name = result.rows[0].slug;
    } else {
      name = appTitle;
    }

    const qq = `update app set current_sequence = $1, name = $2, icon_uri = $3 where id = $4`;
    const vv = [
      sequence,
      name,
      appIcon,
      id,
    ];

    await this.pool.query(qq, vv);
  }

  async createDownstreamVersion(id: string, parentSequence: number, clusterId: string, versionLabel: string, status: string, source: string, diffSummary: string, diffSummaryError: string, commitUrl: string, gitDeployable: boolean): Promise<void> {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");
      let q = `select max(sequence) as last_sequence from app_downstream_version where app_id = $1 and cluster_id = $2`;
      let v: any[] = [
        id,
        clusterId,
      ];
      const result = await pg.query(q, v);
      const newSequence = result.rows[0].last_sequence !== null ? parseInt(result.rows[0].last_sequence) + 1 : 0;

      q = `insert into app_downstream_version (app_id, cluster_id, sequence, parent_sequence, created_at, version_label, status, source, diff_summary, diff_summary_error, git_commit_url, git_deployable) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`;
      v = [
        id,
        clusterId,
        newSequence,
        parentSequence,
        new Date(),
        versionLabel,
        status,
        source,
        diffSummary,
        diffSummaryError,
        commitUrl,
        gitDeployable
      ];
      await pg.query(q, v);
      await pg.query("commit");

    } catch (error) {
      await pg.query("rollback");
      throw error;
    } finally {
      pg.release();
    }
  }

  async getPreviouslyDeployedSequence(appId: string, clusterId: string, currentSequence: number): Promise<number | undefined> {
    const q = `select sequence from app_downstream_version where app_id = $1 and cluster_id = $2 and applied_at is not null order by applied_at desc limit 2`;
    const v = [
      appId,
      clusterId,
    ];

    const result = await this.pool.query(q, v);

    if (result.rowCount !== 2) {
      return;
    }

    return parseInt(result.rows[1].sequence);
  }

  async listPastVersions(appId: string, clusterId: string): Promise<KotsVersion[]> {
    let q = `select current_sequence from app_downstream where app_id = $1 and cluster_id = $2`;
    let v = [
      appId,
      clusterId,
    ];

    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No past versions found`);
    }
    const sequence = result.rows[0].current_sequence;

    // If there is not a current_sequence, then there can't be past versions
    if (sequence === null) {
      return [];
    }

    q =
      `SELECT
         adv.created_at,
         adv.version_label,
         adv.status,
         adv.sequence,
         adv.parent_sequence,
         adv.applied_at,
         adv.source,
         adv.diff_summary,
         adv.diff_summary_error,
         adv.preflight_result,
         adv.preflight_result_created_at,
         adv.git_commit_url,
         adv.git_deployable,
         ado.is_error AS has_error,
         av.kots_installation_spec
        FROM
          app_downstream_version AS adv
        LEFT JOIN
          app_version AS av
        ON
          adv.app_id = av.app_id AND adv.sequence = av.sequence
        LEFT JOIN
          app_downstream_output AS ado
        ON
          adv.app_id = ado.app_id AND adv.cluster_id = ado.cluster_id AND adv.sequence = ado.downstream_sequence
        WHERE
          adv.app_id = $1 AND
          adv.cluster_id = $3 AND
          adv.sequence < $2
        ORDER BY
          adv.sequence DESC`;

    v = [
      appId,
      sequence,
      clusterId
    ];

    result = await this.pool.query(q, v);
    const versionItems: KotsVersion[] = [];

    for (const row of result.rows) {
      const releaseNotes = await this.getReleaseNotes(appId, row.parent_sequence);

      let versionItem: KotsVersion = {
        title: row.version_label,
        status: this.downstreamVersionStatus(row),
        createdOn: row.created_at,
        parentSequence: row.parent_sequence,
        sequence: row.sequence,
        deployedAt: row.applied_at,
        source: row.source,
        diffSummary: row.diff_summary,
        diffSummaryError: row.diff_summary_error,
        releaseNotes: releaseNotes || "",
        preflightResult: row.preflight_result,
        preflightResultCreatedAt: row.preflight_result_created_at,
        commitUrl: row.git_commit_url || "",
        gitDeployable: row.git_deployable
      };
      if (row.kots_installation_spec) {
        try {
          const installationSpec = yaml.safeLoad(row.kots_installation_spec).spec as InstallationSpec;
          versionItem.yamlErrors = installationSpec.yamlErrors;
        } catch (err) {
          console.log(`Failed to unmarshal installation spec yaml for sequence ${versionItem.sequence}`, err);
        }
      }
      versionItems.push(versionItem);
    }

    return versionItems;
  }

  async listPendingVersions(appId: string, clusterId: string): Promise<KotsVersion[]> {
    let q = `select current_sequence from app_downstream where app_id = $1 and cluster_id = $2`;
    let v = [
      appId,
      clusterId,
    ];

    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No pending versions found`);
    }
    let sequence = result.rows[0].current_sequence;

    // If there is not a current_sequence, then all versions are future versions
    if (sequence === null) {
      sequence = -1;
    }

    q = `select adv.created_at, adv.version_label, adv.status, adv.sequence, adv.parent_sequence,
adv.applied_at, adv.source, adv.diff_summary, adv.diff_summary_error, adv.preflight_result, adv.preflight_result_created_at, adv.git_commit_url, adv.git_deployable,
av.kots_installation_spec
from app_downstream_version as adv
left join app_version as av on adv.app_id = av.app_id and adv.sequence = av.sequence
where adv.app_id = $1 and adv.cluster_id = $3 and adv.sequence > $2
order by adv.sequence desc`;

    v = [
      appId,
      sequence,
      clusterId,
    ];

    result = await this.pool.query(q, v);
    const versionItems: KotsVersion[] = [];

    for (const row of result.rows) {
      const releaseNotes = await this.getReleaseNotes(appId, row.parent_sequence);

      let versionItem: KotsVersion = {
        title: row.version_label,
        status: row.status,
        createdOn: row.created_at,
        parentSequence: row.parent_sequence,
        sequence: row.sequence,
        deployedAt: row.applied_at,
        source: row.source,
        diffSummary: row.diff_summary,
        diffSummaryError: row.diff_summary_error,
        releaseNotes: releaseNotes || "",
        preflightResult: row.preflight_result,
        preflightResultCreatedAt: row.preflight_result_created_at,
        commitUrl: row.git_commit_url || "",
        gitDeployable: row.git_deployable
      };
      if (row.kots_installation_spec) {
        try {
          const installationSpec = yaml.safeLoad(row.kots_installation_spec).spec as InstallationSpec;
          versionItem.yamlErrors = installationSpec.yamlErrors;
        } catch (err) {
          console.log(`Failed to unmarshal installation spec yaml for sequence ${versionItem.sequence}`, err);
        }
      }
      versionItems.push(versionItem);
    }

    return versionItems;
  }

  async getGitopsInfo(appId: string, clusterId: string) {
    try {
      const kc = new k8s.KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

      const namespace = process.env["POD_NAMESPACE"]!;

      const secretName = "kotsadm-gitops";
      const secret = await k8sApi.readNamespacedSecret(secretName, namespace);
      const secretData = secret.body.data!;

      const configMapName = "kotsadm-gitops";
      const configmap = await k8sApi.readNamespacedConfigMap(configMapName, namespace);

      const appClusterKey = `${appId}-${clusterId}`;
      if (!(appClusterKey in configmap.body.data!)) {
        throw new ReplicatedError(`No gitops data found for app with id ${appId} and cluster with id ${clusterId}`);
      }

      const base64Data = configmap.body.data![appClusterKey];
      const configMapData = JSON.parse(base64Decode(base64Data));

      let provider = "", publicKey = "", privateKey = "", hostname;
      for (const key of Object.keys(secretData)) {
        const value = base64Decode(secretData[key]);
        if (value === configMapData.repoUri) {
          const index = key.charAt(9);
          provider = base64Decode(secretData[`provider.${index}.type`]);
          publicKey = base64Decode(secretData[`provider.${index}.publicKey`]);
          privateKey = base64Decode(secretData[`provider.${index}.privateKey`]);

          const hostnameKey = `provider.${index}.hostname`;
          if (hostnameKey in secretData) {
            hostname = base64Decode(secretData[hostnameKey]);
          }
          break;
        }
      }

      return {
        provider: provider,
        repoUri: configMapData.repoUri,
        hostname: hostname,
        path: configMapData.path,
        branch: configMapData.branch,
        format: configMapData.format,
        action: configMapData.action,
        publicKey: publicKey,
        privateKey: privateKey,
        lastError: configMapData.lastError
      }
    } catch (err) {
      if (err && err.response && err.response.statusCode !== 404) {
        logger.error(err);
      }
      throw new ReplicatedError(`Failed to get gitops info ${err}`);
    }
  }

  async getDownstreamGitOps(appId: string, clusterId: string): Promise<any> {
    try {
      const gitopsInfo = await this.getGitopsInfo(appId, clusterId);
      return {
        enabled: true,
        provider: gitopsInfo.provider,
        uri: gitopsInfo.repoUri,
        hostname: gitopsInfo.hostname,
        path: gitopsInfo.path,
        branch: gitopsInfo.branch,
        format: gitopsInfo.format,
        action: gitopsInfo.action,
        deployKey: gitopsInfo.publicKey,
        isConnected: gitopsInfo.lastError === "",
      }
    } catch (err) {
      return {
        enabled: false
      };
    }
  }

  async getDeployedVersionBackup(appId: string, clusterId: string): Promise<string|void> {
    const q = `select app_version.backup_spec from app_version
      inner join app_downstream on
        app_version.sequence = app_downstream.current_sequence AND
        app_version.app_id = app_downstream.app_id
      where app_downstream.app_id = $1 and app_downstream.cluster_id = $2`;
    const v = [
      appId,
      clusterId,
    ];
    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      return;
    }
    return result.rows[0].backup_spec;
  }

  async getCurrentVersion(appId: string, clusterId: string): Promise<KotsVersion | undefined> {
    let q = `select current_sequence from app_downstream where app_id = $1 and cluster_id = $2`;
    let v = [
      appId,
      clusterId,
    ];
    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No current downstream version found`);
    }
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select adv.created_at, adv.version_label, adv.status, adv.sequence,
adv.parent_sequence, adv.applied_at, adv.source, adv.diff_summary, adv.diff_summary_error, adv.preflight_result,
adv.preflight_result_created_at, adv.git_commit_url, adv.git_deployable, ado.is_error AS has_error,
av.kots_installation_spec
from app_downstream_version as adv
left join app_version as av on adv.app_id = av.app_id and adv.sequence = av.sequence
left join app_downstream_output as ado
on adv.app_id = ado.app_id and adv.cluster_id = ado.cluster_id and adv.sequence = ado.downstream_sequence
where adv.app_id = $1 and adv.cluster_id = $3 and adv.sequence = $2
order by adv.sequence desc`;

    v = [
      appId,
      sequence,
      clusterId,
    ];

    result = await this.pool.query(q, v);
    const row = result.rows[0];

    if (!row) {
      throw new ReplicatedError(`App Version for clusterId ${clusterId} not found. appId: ${appId}, sequence ${sequence}`);
    }

    const releaseNotes = await this.getReleaseNotes(appId, row.parent_sequence);

    let versionItem: KotsVersion = {
      title: row.version_label,
      status: this.downstreamVersionStatus(row),
      createdOn: row.created_at,
      parentSequence: row.parent_sequence,
      sequence: row.sequence,
      deployedAt: row.applied_at,
      source: row.source,
      diffSummary: row.diff_summary,
      diffSummaryError: row.diff_summary_error,
      releaseNotes: releaseNotes || "",
      preflightResult: row.preflight_result,
      preflightResultCreatedAt: row.preflight_result_created_at,
      commitUrl: row.git_commit_url || "",
      gitDeployable: row.git_deployable
    };
    if (row.kots_installation_spec) {
      try {
        const installationSpec = yaml.safeLoad(row.kots_installation_spec).spec as InstallationSpec;
        versionItem.yamlErrors = installationSpec.yamlErrors;
      } catch (err) {
        console.log(`Failed to unmarshal installation spec yaml for sequence ${versionItem.sequence}`, err);
      }
    }

    return versionItem;
  }

  async getCurrentAppVersion(appId: string): Promise<KotsAppVersion | undefined> {
    let q = `select current_sequence from app where id = $1`;
    let v = [
      appId,
    ];
    let result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No current version found`);
    }
    const sequence = result.rows[0].current_sequence;

    if (sequence === null) {
      return;
    }

    q = `select created_at, version_label, release_notes, status, sequence,
applied_at, backup_spec, kots_installation_spec
from app_version
where app_id = $1 and sequence = $2`;
    v = [
      appId,
      sequence,
    ];

    result = await this.pool.query(q, v);

    if (result.rows.length === 0) {
      throw new ReplicatedError(`No app version found`);
    }

    const row = result.rows[0];

    // There is no parent sequence on midstream versions

    const versionItem: KotsAppVersion = {
      title: row.version_label,
      status: row.status || "",
      createdOn: row.created_at,
      sequence: row.sequence,
      releaseNotes: row.release_notes || "",
      deployedAt: row.applied_at,
      preflightResult: row.preflight_result,
      preflightResultCreatedAt: row.preflight_result_created_at,
      backupSpec: row.backup_spec,
    };

    if (row.kots_installation_spec) {
      try {
        const installationSpec = yaml.safeLoad(row.kots_installation_spec).spec as InstallationSpec;
        versionItem.yamlErrors = installationSpec.yamlErrors;
      } catch (err) {
        console.log(`Failed to unmarshal installation spec yaml for sequence ${versionItem.sequence}`, err);
      }
    }

    return versionItem;
  }

  async getKotsAppSpec(appId: string, sequence: number): Promise<ApplicationSpec | undefined> {
    const q = `select kots_app_spec from app_version where app_id = $1 and sequence = $2`;
    const v = [
      appId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    if (!result.rows.length) {
      return undefined;
    }
    const spec: string = result.rows[0].kots_app_spec;
    if (!spec) {
      return undefined;
    }
    return yaml.safeLoad(spec).spec as ApplicationSpec;
  }

  async getKotsInstallationSpec(appId: string, sequence: number): Promise<InstallationSpec | undefined> {
    const q = `select kots_installation_spec from app_version where app_id = $1 and sequence = $2`;
    const v = [
      appId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    if (!result.rows.length) {
      return undefined;
    }
    const spec: string = result.rows[0].kots_installation_spec;
    if (!spec) {
      return undefined;
    }
    return yaml.safeLoad(spec).spec as InstallationSpec;
  }

  async getAppSpec(appId: string, sequence: number): Promise<string | undefined> {
    const q = `select app_spec from app_version where app_id = $1 and sequence = $2`;
    const v = [
      appId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    if (!result.rows.length) {
      return undefined;
    }
    return result.rows[0].app_spec;
  }

  async updateAppConfigValues(appId: string, sequence: string, configValues: string): Promise<void> {
    const q = `update app_version set config_values = $1 where app_id = $2 and sequence = $3`;
    const v = [
      configValues,
      appId,
      sequence,
    ];
    await this.pool.query(q, v);
  }

  async getAppEncryptionKey(appId: string, sequence: string): Promise<string> {
    const q = `select encryption_key from app_version where app_id = $1 and sequence = $2`;
    const v = [
      appId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    const rows = result.rows;

    if (rows.length === 0) {
      throw new ReplicatedError("App not found while trying to get the encryption key");
    }

    if (!rows[0].encryption_key) {
      return "";
    }

    return rows[0].encryption_key;
  }

  async getMaxSequence(appId: string): Promise<number> {
    const q = `select max(sequence) as sequence from app_version where app_id = $1`;
    const v = [
      appId,
    ];

    const result = await this.pool.query(q, v);
    const rows = result.rows;

    if (rows.length === 0 || !rows[0].sequence) {
      return 0;
    }

    return parseInt(rows[0].sequence);
  }

  async getMidstreamUpdateCursor(appId: string): Promise<UpdateCursor> {
    const q = `select update_cursor, channel_name from app_version where app_id = $1 order by sequence desc limit 1`;
    const v = [
      appId,
    ];

    const result = await this.pool.query(q, v);

    if (result.rows.length === 0) {
      return {
        cursor: "",
        channelName: "",
      };
    }

    return {
      cursor: result.rows[0].update_cursor,
      channelName: result.rows[0].channel_name,
    };
  }

  async getReleaseNotes(appId: string, sequence: number): Promise<string | undefined> {
    const q = `SELECT release_notes FROM app_version WHERE app_id = $1 AND sequence = $2`;
    const v = [
      appId,
      sequence
    ];
    const result = await this.pool.query(q, v);
    const row = result.rows[0];

    return row && row.release_notes;
  }

  async updateFailedInstallState(appSlug: string): Promise<Boolean> {
    const q = `update app set install_state = $1 where slug = $2`;
    const v = ["failed", appSlug];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`No app with the slug of ${appSlug} was found`);
    }

    return true;
  }

  async deployVersion(appId: string, sequence: number): Promise<void> {
    const q = `update app_downstream set current_sequence = $1 where app_id = $2`;
    const v = [
      sequence,
      appId,
    ];
    await this.pool.query(q, v);

    const qq = `UPDATE app_downstream_version
        SET status = 'deployed', applied_at = $3
      WHERE sequence = $1 AND app_id = $2`;

    const vv = [
      sequence,
      appId,
      new Date()
    ];

    await this.pool.query(qq, vv);
  }

  async getAppRegistryDetails(appId: string, maskPassword?: boolean): Promise<KotsAppRegistryDetails> {
    const q = `select registry_hostname, registry_username, registry_password, registry_password_enc, namespace, last_registry_sync from app where id = $1`;
    const v = [
      appId,
    ];
    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`Unable to get registry details for app with the ID of ${appId}`);
    }

    const regInfo = this.mapAppRegistryDetails(result.rows[0]);
    await this.migrationEncryptRegistryCredentials(appId, regInfo);
    await this.decryptRegistryCredentials(appId, regInfo);

    if (maskPassword) {
      regInfo.registryPassword = this.getPasswordMask();
    }

    return regInfo
  }

  getPasswordMask(): string {
    return "***HIDDEN***";
  }

  async decryptRegistryCredentials(appId: string, regInfo: KotsAppRegistryDetails) {
    if (!this.params.apiEncryptionKey) {
      return regInfo;
    }

    if (regInfo.registryPassword && regInfo.registryPassword.length > 0) {
      return regInfo;
    }

    if (!regInfo.registryPasswordEnc || regInfo.registryPasswordEnc.length === 0) {
      return regInfo
    }

    regInfo.registryPassword = await kotsDecryptString(this.params.apiEncryptionKey, regInfo.registryPasswordEnc)

    return regInfo
  }

  async migrationEncryptRegistryCredentials(appId: string, regInfo: KotsAppRegistryDetails) {
    if (!this.params.apiEncryptionKey) {
      return regInfo;
    }

    if (!regInfo.registryPassword || regInfo.registryPassword.length === 0) {
      return regInfo;
    }

    if (regInfo.registryPasswordEnc && regInfo.registryPasswordEnc.length > 0) {
      return regInfo
    }

    regInfo.registryPasswordEnc = await kotsEncryptString(this.params.apiEncryptionKey, regInfo.registryPassword);
    const q = `update app set registry_password = NULL, registry_password_enc = $1 where id = $2`;
    const v = [
      regInfo.registryPasswordEnc,
      appId,
    ];
    await this.pool.query(q, v);

    return regInfo
  }

  async updateRegistryDetails(appId: string, hostname: string, username: string, password: string, namespace: string): Promise<void> {
    let q: string;
    let v: any;

    if (password === this.getPasswordMask()) {
      q = `update app set registry_hostname = $1, registry_username = $2, registry_password = NULL, namespace = $3, last_registry_sync = $4 where id = $5`;
      v = [
        hostname,
        username,
        namespace,
        new Date(),
        appId,
      ];
    } else if (this.params.apiEncryptionKey) {
      const passwordEnc = await kotsEncryptString(this.params.apiEncryptionKey, password);
      q = `update app set registry_hostname = $1, registry_username = $2, registry_password = NULL, registry_password_enc = $3, namespace = $4, last_registry_sync = $5 where id = $6`;
      v = [
        hostname,
        username,
        passwordEnc,
        namespace,
        new Date(),
        appId,
      ];
    } else {
      q = `update app set registry_hostname = $1, registry_username = $2, registry_password = $3, namespace = $4, last_registry_sync = $5 where id = $6`;
      v = [
        hostname,
        username,
        password,
        namespace,
        new Date(),
        appId,
      ];
    }
    await this.pool.query(q, v);
  }

  async listInstalledKotsApps(userId?: string): Promise<KotsApp[]> {
    const q = `select id from app inner join user_app on app_id = id where user_app.user_id = $1 and install_state = 'installed'`;
    const v = [userId];

    const result = await this.pool.query(q, v);
    const apps: KotsApp[] = [];
    for (const row of result.rows) {
      apps.push(await this.getApp(row.id));
    }

    const qq = `select id from app where is_all_users = true and install_state = 'installed'`;
    const resultTwo = await this.pool.query(qq);
    for (const row of resultTwo.rows) {
      apps.push(await this.getApp(row.id));
    }

    return apps;
  }

  async getPendingKotsAirgapApp(): Promise<KotsApp> {
    const q = `select id from app where install_state in ('airgap_upload_pending', 'airgap_upload_in_progress', 'airgap_upload_error')`;
    const v = [];

    const result = await this.pool.query(q, v);
    if (result.rows.length === 0) {
      throw new ReplicatedError(`No pending airgap apps found`);
    }
    if (result.rows.length > 1) {
      throw new ReplicatedError(`Airgap install is not allowed`);
    }

    const app = await this.getApp(result.rows[0].id);
    return app;
  }

  async setKotsAppInstallState(appId: string, state: string) {
    const q = `update app set install_state = $1 where id = $2`;
    const v = [
      state,
      appId
    ];

    await this.pool.query(q, v);
  }

  async setKotsAirgapAppInstalled(appId: string) {
    const q = `update app set install_state = 'installed', is_airgap = true where id = $1`;
    const v = [appId];

    await this.pool.query(q, v);
  }

  async deleteDownstream(appId: string, clusterId: string): Promise<Boolean> {
    const q = `delete from app_downstream where app_id = $1 and cluster_id = $2`;
    const v = [appId, clusterId];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`No downstreams with the id of ${clusterId} were found`);
    }

    return true;
  }

  async deleteApp(appId: string): Promise<Boolean> {
    const pg = await this.pool.connect();
    try {
      let q: string;
      const v = [appId];

      await pg.query("begin");
      q = `delete from user_app where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app_version where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app_downstream where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app_downstream_version where app_id = $1`;
      await pg.query(q, v);

      q = `delete from app where id = $1`;
      await pg.query(q, v);

      await pg.query("commit");
    } finally {
      await pg.query("rollback");
      pg.release();
    }
    return true;
  }

  async getApp(id: string): Promise<KotsApp> {
    const q = `select id, name, license, upstream_uri, icon_uri, created_at, updated_at, slug, current_sequence, last_update_check_at, is_airgap, snapshot_ttl_new, snapshot_schedule, restore_in_progress_name, restore_undeploy_status, update_checker_spec from app where id = $1`;
    const v = [id];

    const result = await this.pool.query(q, v);

    if (result.rowCount == 0) {
      throw new ReplicatedError("not found");
    }
    const row = result.rows[0];

    const current_sequence = row.current_sequence;
    const qq = `SELECT preflight_spec, config_spec FROM app_version WHERE app_id = $1 AND sequence = $2`;

    const vv = [
      id,
      current_sequence
    ];

    const rr = await this.pool.query(qq, vv);
    const kotsApp = new KotsApp();
    kotsApp.id = row.id;
    kotsApp.name = row.name;
    kotsApp.license = row.license;
    kotsApp.isAirgap = row.is_airgap;
    kotsApp.upstreamUri = row.upstream_uri;
    kotsApp.iconUri = row.icon_uri;
    kotsApp.createdAt = new Date(row.created_at);
    kotsApp.updatedAt = row.updated_at ? new Date(row.updated_at) : undefined;
    kotsApp.slug = row.slug;
    kotsApp.currentSequence = row.current_sequence;
    kotsApp.lastUpdateCheckAt = row.last_update_check_at ? new Date(row.last_update_check_at) : undefined;
    kotsApp.bundleCommand = await kotsApp.getSupportBundleCommand(row.slug);
    // This is to avoid a race condition when uploading a license file where the row in app_version
    // has not been created yet
    kotsApp.hasPreflight = !!rr.rows[0] && !!rr.rows[0].preflight_spec;
    kotsApp.isConfigurable = !!rr.rows[0] && !!rr.rows[0].config_spec;
    kotsApp.snapshotTTL = row.snapshot_ttl_new;
    kotsApp.snapshotSchedule = row.snapshot_schedule;
    kotsApp.restoreInProgressName = row.restore_in_progress_name;
    kotsApp.restoreUndeployStatus = row.restore_undeploy_status;
    kotsApp.updateCheckerSpec = row.update_checker_spec;

    return kotsApp;
  }

  async getKotsAppLicenseType(appId: string, sequence: number): Promise<string> {
    const q = `select kots_license from app_version where app_id = $1 and sequence = $2`;
    const v = [
      appId,
      sequence,
    ];

    const result = await this.pool.query(q, v);
    if (result.rowCount == 0) {
      return "";
    }

    const row = result.rows[0];
    const license: string = row.kots_license;
    if (!license) {
      return "";
    }

    try {
      const licenseType = yaml.safeLoad(license).spec.licenseType;
      return licenseType;
    } catch (err) {
      console.log(err);
      return "";
    }
  }

  async isGitOpsSupported(appId: string, sequence: number): Promise<boolean> {
    const kc = new k8s.KubeConfig();
    kc.loadFromDefault();
    const k8sApi = kc.makeApiClient(k8s.CoreV1Api);

    try {
      const namespace = process.env["POD_NAMESPACE"]!;
      const secretName = "kotsadm-gitops";
      await k8sApi.readNamespacedSecret(secretName, namespace);
      return true;
    } catch (err) {
      // secret does not exist
    }

    const q = `select kots_license from app_version where app_id = $1 and sequence = $2`;
    const v = [
      appId,
      sequence,
    ];

    const result = await this.pool.query(q, v);

    if (result.rowCount == 0) {
      throw new ReplicatedError("No app versions found");
    }

    const row = result.rows[0];
    const license: string = row.kots_license;
    if (!license) {
      return true;
    }

    try {
      return !!yaml.safeLoad(license).spec.isGitOpsSupported;
    } catch (err) {
      console.log(err);
      return false;
    }
  }

  async getIdFromSlug(slug: string): Promise<string> {
    const q = "select id from app where slug = $1";
    const v = [slug];

    const result = await this.pool.query(q, v);
    if (result.rowCount === 0) {
      throw new ReplicatedError(`Unable to find appId for slug ${slug}`);
    }
    return result.rows[0].id;
  }

  async createKotsApp(name: string, upstreamURI: string, license: string, airgapEnabled: boolean, userId?: string): Promise<KotsApp> {
    if (!name) {
      throw new Error("missing name");
    }

    const id = randomstring.generate({ capitalization: "lowercase" });
    const titleForSlug = name.replace(/\./g, "-");

    let slugProposal = slugify(titleForSlug, { lower: true });

    let i = 0;
    let foundUniqueSlug = false;
    while (!foundUniqueSlug) {
      if (i > 0) {
        slugProposal = `${slugify(titleForSlug, { lower: true })}-${i}`;
      }
      const qq = `select count(1) as count from app where slug = $1`;
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

    let installState: string;
    if (!upstreamURI.toLowerCase().startsWith("replicated://")) {
      installState = "installed";
    } else {
      installState = airgapEnabled ? "airgap_upload_pending" : "online_upload_pending";
    }
    try {
      await pg.query("begin");
      const q = `insert into app (id, name, icon_uri, created_at, slug, upstream_uri, license, is_all_users, install_state)
      values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`;
      const v = [
        id,
        name,
        "",
        new Date(),
        slugProposal,
        upstreamURI,
        license,
        !userId,
        installState,
      ];

      await pg.query(q, v);

      if (userId) { // unset user id means all users
        const uwq = "insert into user_app (user_id, app_id) values ($1, $2)";
        const uwv = [userId, id];
        await pg.query(uwq, uwv);
      }

      await pg.query("commit");
      const app = await this.getApp(id);

      return app;
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async updateKotsAppLicense(appId: string, license: string): Promise<void> {
    const q = `update app set license = $1 where id = $2`;
    const v = [license, appId];
    await this.pool.query(q, v);
  }

  async updateApp(id: string, appName?: string, iconUri?: string) {
    const pg = await this.pool.connect();

    try {
      await pg.query("begin");

      if (appName) {
        const q = "UPDATE app SET name = $2 WHERE id = $1";
        const v = [id, appName];
        await pg.query(q, v);
      }

      if (iconUri) {
        const q = "UPDATE app SET icon_uri = $2 WHERE id = $1";
        const v = [id, iconUri];
        await pg.query(q, v);
      }

      await pg.query("commit");
    } finally {
      await pg.query("rollback");
      pg.release();
    }
  }

  async updateAppSnapshotTTL(appId: string, snapshotTTL: string): Promise<void> {
    const q = `update app set snapshot_ttl_new = $1 where id = $2`;
    const v = [snapshotTTL, appId];
    await this.pool.query(q, v);
  }

  async updateAppSnapshotSchedule(appId: string, snapshotSchedule: string|null): Promise<void> {
    const q = `update app set snapshot_schedule = $1 where id = $2`;
    const v = [snapshotSchedule, appId];
    await this.pool.query(q, v);
  }

  async updateAppRestoreInProgressName(appId: string, restoreInProgressName: string): Promise<void> {
    const q = `update app set restore_in_progress_name = $1 where id = $2`;
    const v = [restoreInProgressName, appId];
    await this.pool.query(q, v);
  }

  async updateAppRestoreUndeployStatus(appId: string, undeployStatus: string): Promise<void> {
    const q = `update app set restore_undeploy_status = $1 where id = $2`;
    const v = [undeployStatus, appId];
    await this.pool.query(q, v);
  }

  async updateAppRestoreReset(appId): Promise<void> {
    const q = `update app set restore_in_progress_name = NULL, restore_undeploy_status = '' where id = $1`;
    const v = [appId];
    await this.pool.query(q, v);
  }

  async ignorePreflightPermissionErrors(appId: string, clusterId: string, sequence: number): Promise<void> {
    const q = `UPDATE app_downstream_version
SET status = 'pending_preflight', preflight_ignore_permissions = true, preflight_result = null
WHERE app_id = $1 AND cluster_id = $2 AND sequence = $3`;

    const v = [
      appId,
      clusterId,
      sequence
    ];

    await this.pool.query(q, v);
  }

  async retryPreflights(appId: string, clusterId: string, sequence: number): Promise<void> {
    const q = `UPDATE app_downstream_version
SET status = 'pending_preflight', preflight_ignore_permissions = false, preflight_result = null
WHERE app_id = $1 AND cluster_id = $2 AND sequence = $3`;

    const v = [
      appId,
      clusterId,
      sequence
    ];

    await this.pool.query(q, v);
  }

  async getAirgapInstallStatus(): Promise<{ installStatus: string, currentMessage: string }> {
    const q = `SELECT install_state from app ORDER BY created_at DESC LIMIT 1`;
    const result = await this.pool.query(q);

    if (result.rows.length !== 1) {
      return {
        installStatus: "not_installed",
        currentMessage: "",
      }
    }

    const taskStatus = await this.getApiTaskStatus("airgap-install");

    return {
      installStatus: result.rows[0].install_state,
      currentMessage: taskStatus.currentMessage,
    };
  }

  async getOnlineInstallStatus(): Promise<{ installStatus: string, currentMessage: string }> {
    const q = `SELECT install_state from app ORDER BY created_at DESC LIMIT 1`;
    const result = await this.pool.query(q);

    if (result.rows.length !== 1) {
      return {
        installStatus: "not_installed",
        currentMessage: "",
      }
    }

    const taskStatus = await this.getApiTaskStatus("online-install");

    return {
      installStatus: result.rows[0].install_state,
      currentMessage: taskStatus.currentMessage,
    };
  }

  async clearAirgapInstallInProgress(): Promise<void> {
    await this.clearApiTaskStatus("airgap-install");
  }

  async clearOnlineInstallInProgress(): Promise<void> {
    await this.clearApiTaskStatus("online-install");
  }

  async setAirgapInstallStatus(msg: string, status: string): Promise<void> {
    await this.setApiTaskStatus("airgap-install", msg, status);
  }

  async setOnlineInstallStatus(msg: string, status: string): Promise<void> {
    await this.setApiTaskStatus("online-install", msg, status);
  }

  async updateAirgapInstallLiveness(): Promise<void> {
    await this.updateApiTaskStatusLiveness("airgap-install");
  }

  async updateOnlineInstallLiveness(): Promise<void> {
    await this.updateApiTaskStatusLiveness("online-install");
  }

  async setAirgapInstallFailed(appId: string): Promise<void> {
    const q = `update app set install_state = 'airgap_upload_error' where id = $1`;
    const v = [appId];
    await this.pool.query(q, v);
  }

  async setOnineInstallFailed(appId: string): Promise<void> {
    const q = `update app set install_state = 'online_upload_error' where id = $1`;
    const v = [appId];
    await this.pool.query(q, v);
  }

  async resetAirgapInstallInProgress(appId: string): Promise<void> {
    const q = `update app set install_state = 'airgap_upload_in_progress' where id = $1`;
    const v = [appId];
    await this.pool.query(q, v);

    await this.clearApiTaskStatus("airgap-install");
  }

  async resetOnlineInstallInProgress(appId: string): Promise<void> {
    const q = `update app set install_state = 'online_upload_in_progress' where id = $1`;
    const v = [appId];
    await this.pool.query(q, v);

    await this.clearApiTaskStatus("online-install");
  }

  async getImageRewriteStatus(): Promise<{ currentMessage: string, status: string }> {
    return this.getApiTaskStatus("image-rewrite");
  }

  async setImageRewriteStatus(msg: string, status: string): Promise<void> {
    await this.setApiTaskStatus("image-rewrite", msg, status);
  }

  async clearImageRewriteStatus(): Promise<void> {
    await this.clearApiTaskStatus("image-rewrite");
  }

  async updateImageRewriteStatusLiveness(): Promise<void> {
    await this.updateApiTaskStatusLiveness("image-rewrite");
  }

  async setUpdateDownloadStatus(msg: string, status: string): Promise<void> {
    await this.setApiTaskStatus("update-download", msg, status);
  }

  async clearUpdateDownloadStatus(): Promise<void> {
    await this.clearApiTaskStatus("update-download");
  }

  async updateUpdateDownloadStatusLiveness(): Promise<void> {
    await this.updateApiTaskStatusLiveness("update-download");
  }

  async setApiTaskStatus(id: string, msg: string, status: string): Promise<void> {
    const q = `insert into api_task_status (id, updated_at, current_message, status) values ($1, $2, $3, $4)
    on conflict(id) do update set current_message = EXCLUDED.current_message, status = EXCLUDED.status`;
    const v = [id, new Date(), msg, status];
    await this.pool.query(q, v);
  }

  async getApiTaskStatus(id: string): Promise<{ currentMessage: string, status: string }> {
    // status older than <N> seconds is considered stale as it should be updated once per second
    const q = `select status, current_message from api_task_status where id = $1 AND updated_at > ($2::timestamp - '10 seconds'::interval)`;
    const result = await this.pool.query(q, [id, new Date()]);

    if (result.rows.length !== 1) {
      return {
        currentMessage: "",
        status: "",
      };
    }
    return {
      currentMessage: result.rows[0].current_message,
      status: result.rows[0].status,
    };
  }

  async clearApiTaskStatus(id: string): Promise<void> {
    const q = `delete from api_task_status where id = $1`;
    await this.pool.query(q, [id]);
  }

  async updateApiTaskStatusLiveness(id: string): Promise<void> {
    const q = `update api_task_status set updated_at = $1 where id = $2`;
    const v = [new Date(), id];
    await this.pool.query(q, v);
  }

  private downstreamVersionStatus(row: any): string {
    let status = "unknown";

    // first check if operator has reported back.
    // and if it hasn't, we should not show "deployed" to the user.

    if (row.has_error === false) {
      status = row.status;
    } else if (row.has_error === true) {
      status = "failed";
    } else if (row.status === "deployed") {
      status = "deploying";
    } else if (row.status) {
      status = row.status;
    }

    return status;
  }

  private mapAppRegistryDetails(row: any): KotsAppRegistryDetails {
    if (!row) {
      throw new ReplicatedError("No app provided to map function");
    }
    return {
      registryHostname: row.registry_hostname,
      registryUsername: row.registry_username,
      registryPassword: row.registry_password,
      registryPasswordEnc: row.registry_password_enc,
      namespace: row.namespace,
      lastSyncedAt: row.last_registry_sync,
    };
  }

}
