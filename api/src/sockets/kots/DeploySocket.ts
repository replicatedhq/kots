import { IO, Nsp, SocketService, SocketSession, Socket } from "@tsed/socketio";
import { getPostgresPool } from "../../util/persistence/db";
import { KotsAppStore } from "../../kots_app/kots_app_store";
import { KotsAppStatusStore } from "../../kots_app/kots_app_status_store";
import { State } from "../../kots_app";
import { Params } from "../../server/params";
import { ClusterStore } from "../../cluster";
import { PreflightStore } from "../../preflight/preflight_store";
import _ from "lodash";
import { TroubleshootStore } from "../../troubleshoot";
import {logger} from "../../server/logger";
import { VeleroClient } from "../../snapshots/resolvers/veleroClient";
import { Phase } from "../../snapshots/velero";

const DefaultReadyState = [{kind: "EMPTY", name: "EMPTY", namespace: "EMPTY", state: State.Ready}];

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

  // tslint:disable-next-line cyclomatic-complexity
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

        switch (app.restoreUndeployStatus) {
        case "in_process":
          // undeploy in process, continue loop
          break;

        case "completed":
          logger.info(`Restore successfully removed current app version.`);

          let parts = app.restoreInProgressName.split("-");
          parts = parts.slice(0, parts.length-1); // trim restore time to get snapshot name
          const snapshotName = parts.join("-");

          const velero = new VeleroClient("velero"); // TODO velero namespace

          try {
            const restore = await velero.readRestore(app.restoreInProgressName);
            if (!restore) {
              // create the Restore resource
              await velero.restore(snapshotName, app.restoreInProgressName);
              logger.info(`Created Restore object ${app.restoreInProgressName}`);
            }
          } catch (err) {
            console.log("Velero restore failed");
            console.log(err);
          }
          break;

        case "failed":
          logger.warn(`Restore ${app.restoreInProgressName} falied`);
          // TODO
          break;

        default:
          const deployedAppVersion = await this.kotsAppStore.getCurrentVersion(app.id, clusterSocketHistory.clusterId);
          const maybeDeployedAppSequence = deployedAppVersion && deployedAppVersion.sequence;
          if (maybeDeployedAppSequence! > -1) {
            const deployedAppSequence = Number(maybeDeployedAppSequence);
            const cluster = await this.clusterStore.getCluster(clusterSocketHistory.clusterId);
            try {
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

              this.io.in(clusterSocketHistory.clusterId).emit("deploy", args);

              // reset app deployment state
              clusterSocketHistory.sentDeploySequences = _.filter(clusterSocketHistory.sentDeploySequences, (s) => {
                return !_.startsWith(s, app.id);
              });

              await this.kotsAppStore.updateAppRestoreUndeployStatus(app.id, "in_process");
            } catch (err) {
              console.log("Restore undeploy failed");
              console.log(err);
            }
            break;
          }
        }
      }
    }
  }

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
