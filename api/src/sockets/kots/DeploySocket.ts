import _ from "lodash";
import { IO, Nsp, SocketService, SocketSession, Socket } from "@tsed/socketio";
import { getPostgresPool } from "../../util/persistence/db";
import { KotsAppStore, UndeployStatus } from "../../kots_app/kots_app_store";
import { KotsAppStatusStore } from "../../kots_app/kots_app_status_store";
import { State, KotsApp } from "../../kots_app";
import { Params } from "../../server/params";
import { ClusterStore, Cluster } from "../../cluster";
import { PreflightStore } from "../../preflight/preflight_store";
import { TroubleshootStore } from "../../troubleshoot";
import { logger } from "../../server/logger";
import { VeleroClient } from "../../snapshots/resolvers/veleroClient";
import { kotsAppSequenceKey, kotsClusterIdKey } from "../../snapshots/snapshot";
import { Phase, Restore } from "../../snapshots/velero";
import { ReplicatedError } from "../../server/errors";

const DefaultReadyState = [{kind: "EMPTY", name: "EMPTY", namespace: "EMPTY", state: State.Ready}];

const oneMinuteInMilliseconds = 1 * 60 * 1000;

interface ClusterSocketHistory {
  clusterId: string;
  socketId: string;
  sentPreflightUrls: {[key: string]: boolean};
  sentDeploySequences: string[];
}


@SocketService("")
export class KotsDeploySocketService {
  @Nsp nsp: SocketIO.Namespace;
  kotsAppStore: KotsAppStore;
  kotsAppStatusStore: KotsAppStatusStore;
  clusterStore: ClusterStore;
  preflightStore: PreflightStore;
  troubleshootStore: TroubleshootStore;
  clusterSocketHistory: ClusterSocketHistory[];
  params: Params;
  lastUndeployTime: number = 0;

  constructor(@IO private io: SocketIO.Server) {
    getPostgresPool()
      .then((pool) => {
        Params.getParams()
          .then((params) => {
            this.params = params;
            this.kotsAppStore = new KotsAppStore(pool, params);
            this.kotsAppStatusStore = new KotsAppStatusStore(pool, params);
            this.clusterStore = new ClusterStore(pool, params);
            this.preflightStore = new PreflightStore(pool);
            this.troubleshootStore = new TroubleshootStore(pool, params);
            this.clusterSocketHistory = [];

            setInterval(this.preflightLoop.bind(this), 1000);
            setInterval(this.supportBundleLoop.bind(this), 1000);
            setInterval(this.restoreLoop.bind(this), 1000);
          })
      });
  }

  /**
   * Triggered when a new client connects to the Namespace.
   */
  async $onConnection(@Socket socket: SocketIO.Socket, @SocketSession session: SocketSession) {
    if (!this.clusterStore) {
      // we aren't ready
      socket.disconnect();
      return;
    }

    const cluster = await this.clusterStore.getFromDeployToken(socket.handshake.query.token);
    console.log(`Cluster ${cluster.id} joined`);
    socket.join(cluster.id);

    this.clusterSocketHistory.push({
      clusterId: cluster.id,
      socketId: socket.id,
      sentPreflightUrls: {},
      sentDeploySequences: [],
    });
  }

  /**
   * Triggered when a client disconnects from the Namespace.
   */
  $onDisconnect(@Socket socket: SocketIO.Socket) {
    const updated = _.reject(this.clusterSocketHistory, (csh) => {
      return csh.socketId === socket.id;
    });
    this.clusterSocketHistory = updated;
  }

  async supportBundleLoop() {
    if (!this.clusterSocketHistory) {
      return;
    }

    for (const clusterSocketHistory of this.clusterSocketHistory) {
      const pendingSupportBundles = await this.troubleshootStore.listPendingSupportBundlesForCluster(clusterSocketHistory.clusterId);
      for (const pendingSupportBundle of pendingSupportBundles) {
        const app = await this.kotsAppStore.getApp(pendingSupportBundle.appId);
        this.io.in(clusterSocketHistory.clusterId).emit("supportbundle", {uri: `${this.params.shipApiEndpoint}/api/v1/troubleshoot/${app.slug}?incluster=true`});
        await this.troubleshootStore.clearPendingSupportBundle(pendingSupportBundle.id);
      }
    }
  }

  async restoreLoop() {
    if (!this.clusterSocketHistory) {
      return;
    }

    for (const clusterSocketHistory of this.clusterSocketHistory) {
      const apps = await this.kotsAppStore.listAppsForCluster(clusterSocketHistory.clusterId);
      for (const app of apps) {
        if (!app.restoreInProgressName) {
          continue;
        }

        const cluster = await this.clusterStore.getCluster(clusterSocketHistory.clusterId);
        try {
          await this.handleRestoreInProgress(app, cluster);
        } catch (err) {
          logger.warn("Failed to handle restore in progress");
          logger.warn(err);
        }
      }
    }
  }

  async handleRestoreInProgress(app: KotsApp, cluster: Cluster): Promise<void> {
    if (!app.restoreInProgressName) {
      return;
    }
  
    switch (app.restoreUndeployStatus) {
    case UndeployStatus.InProcess:
      // retry undeploy every minute since socket.io is not bi-directional
      const lastUndeployInterval = new Date().getTime() - this.lastUndeployTime;
      if (lastUndeployInterval >= oneMinuteInMilliseconds) {
        await this.undeployApp(app, cluster);
        this.lastUndeployTime = new Date().getTime();
      }
      break;
  
    case UndeployStatus.Completed:
      await this.handleUndeployCompleted(app, cluster);
      break;
  
    case UndeployStatus.Failed:
      logger.warn(`Restore ${app.restoreInProgressName} falied`);
      // TODO
      break;
  
    default:
      // start undeploy
      await this.undeployApp(app, cluster);
      this.lastUndeployTime = new Date().getTime();
    }
  }
  
  async undeployApp(app: KotsApp, cluster: Cluster): Promise<void> {
    logger.info(`Starting restore, undeploying app ${app.name}`);

    const desiredNamespace = ".";
    const rendered = await app.render(app.currentSequence!.toString(), `overlays/downstreams/${cluster.title}`);
    const b = new Buffer(rendered);

    const kotsAppSpec = await app.getKotsAppSpec(cluster.id, this.kotsAppStore);

    // make operator prune everything
    const args = {
      app_id: app.id,
      kubectl_version: kotsAppSpec ? kotsAppSpec.kubectlVersion : "",
      namespace: desiredNamespace,
      manifests: "",
      previous_manifests: b.toString("base64"),
      result_callback: "/api/v1/undeploy/result",
      wait: true,
    };

    this.io.in(cluster.id).emit("deploy", args);

    await this.kotsAppStore.updateAppRestoreUndeployStatus(app.id, UndeployStatus.InProcess);
  }

  async handleUndeployCompleted(app: KotsApp, cluster: Cluster): Promise<void> {
    if (!app.restoreInProgressName) {
      return;
    }

    const velero = new VeleroClient("velero"); // TODO velero namespace
    const restore = await velero.readRestore(app.restoreInProgressName);
    if (!restore) {
      await this.startVeleroRestore(velero, app);
    } else {
      await this.checkRestoreComplete(velero, restore, app);
    }
  }

  async startVeleroRestore(velero: VeleroClient, app: KotsApp): Promise<void> {
    if (!app.restoreInProgressName) {
      return;
    }

    logger.info(`Creating velero Restore object ${app.restoreInProgressName}`);

    // create the Restore resource
    const snapshotName = getSnapshotNameFromRestoreName(app.restoreInProgressName);
    await velero.restore(snapshotName, app.restoreInProgressName);
  }

  async checkRestoreComplete(velero: VeleroClient, restore: Restore, app: KotsApp) {
    switch (_.get(restore, "status.phase")) {
      case Phase.Completed:
        // Switch operator back to deploy mode on the restored sequence
        const backup = await velero.readBackup(restore.spec.backupName);
        if (!backup.metadata.annotations) {
          throw new ReplicatedError(`Backup is missing required annotations`);
        }
        const sequenceString = backup.metadata.annotations[kotsAppSequenceKey];
        if (!sequenceString) {
          throw new ReplicatedError(`Backup is missing sequence annotation`);
        }
        const sequence = parseInt(sequenceString, 10);
        if (_.isNaN(sequence)) {
          throw new ReplicatedError(`Failed to parse sequence from Backup: ${sequenceString}`);
        }
        const clusterId = backup.metadata.annotations[kotsClusterIdKey];
        if (!clusterId) {
          throw new ReplicatedError(`Backup is missing cluster ID annotation`);
        }

        logger.info(`Restore complete, setting deploy version to ${sequence}`);
        await this.kotsAppStore.deployVersion(app.id, sequence, clusterId);
        await this.kotsAppStore.updateAppRestoreReset(app.id);
        break;

      case Phase.PartiallyFailed:
      case Phase.Failed:
        logger.info(`Restore failed, resetting app restore name`);
        await this.kotsAppStore.updateAppRestoreReset(app.id);
        break;

      default:
        // in progress
    }
  }

  // tslint:disable-next-line cyclomatic-complexity
  async preflightLoop() {
    if (!this.clusterSocketHistory) {
      return;
    }

    for (const clusterSocketHistory of this.clusterSocketHistory) {
      const apps = await this.kotsAppStore.listAppsForCluster(clusterSocketHistory.clusterId);
      for (const app of apps) {
        if (app.restoreInProgressName) {
          continue;
        }
        const pendingPreflightParams = await this.preflightStore.getPendingPreflightParams(true);
        for (const param of pendingPreflightParams) {
          if (clusterSocketHistory.sentPreflightUrls[param.url] !== param.ignorePermissions) {
            const msg = {
              uri: param.url,
              ignorePermissions: param.ignorePermissions
            }
            this.io.in(clusterSocketHistory.clusterId).emit("preflight", msg);
            clusterSocketHistory.sentPreflightUrls[param.url] = param.ignorePermissions;
          }
        }

        const deployedAppVersion = await this.kotsAppStore.getCurrentVersion(app.id, clusterSocketHistory.clusterId);
        const maybeDeployedAppSequence = deployedAppVersion && deployedAppVersion.sequence;
        if (maybeDeployedAppSequence! > -1) {
          const deployedAppSequence = Number(maybeDeployedAppSequence);
          if (clusterSocketHistory.sentDeploySequences.indexOf(`${app.id}/${deployedAppSequence}`) === -1) {
            const cluster = await this.clusterStore.getCluster(clusterSocketHistory.clusterId);
            try {
              const desiredNamespace = ".";
              const rendered = await app.render(app.currentSequence!.toString(), `overlays/downstreams/${cluster.title}`);
              const b = new Buffer(rendered);

              const kotsAppSpec = await app.getKotsAppSpec(cluster.id, this.kotsAppStore);

              const args = {
                app_id: app.id,
                kubectl_version: kotsAppSpec ? kotsAppSpec.kubectlVersion : "",
                namespace: desiredNamespace,
                manifests: b.toString("base64"),
                previous_manifests: "",
                result_callback: "/api/v1/deploy/result",
                wait: false,
              };

              const previousSequence = await this.kotsAppStore.getPreviouslyDeployedSequence(app.id, clusterSocketHistory.clusterId, deployedAppSequence);
              if (previousSequence !== undefined) {
                const previousRendered = await app.render(previousSequence.toString(), `overlays/downstreams/${cluster.title}`);
                const bb = new Buffer(previousRendered);
                args.previous_manifests = bb.toString("base64");
              }

              this.io.in(clusterSocketHistory.clusterId).emit("deploy", args);
              clusterSocketHistory.sentDeploySequences.push(`${app.id}/${deployedAppSequence}`);
            } catch(err) {
              await this.kotsAppStore.updateDownstreamsStatus(app.id, deployedAppSequence, "failed", String(err));
              continue;
            }

            try {
              const kotsAppSpec = await app.getKotsAppSpec(cluster.id, this.kotsAppStore)
              if (kotsAppSpec && kotsAppSpec.statusInformers) {
                this.io.in(clusterSocketHistory.clusterId).emit("appInformers", {
                  app_id: app.id,
                  informers: kotsAppSpec.statusInformers,
                });
              } else {
                // no informers, set state to ready
                await this.kotsAppStatusStore.setKotsAppStatus(app.id, DefaultReadyState, new Date());
              }
            } catch (err) {
              console.log(err);
            }
          }
        }
      }
    }
  }
}
  
function getSnapshotNameFromRestoreName(restoreName: string): string {
  const parts = restoreName.split("-");
  return parts.slice(0, parts.length-1).join("-");
}
