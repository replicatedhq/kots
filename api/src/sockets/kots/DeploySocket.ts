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

const DefaultReadyState = JSON.stringify([{state: State.Ready}]);

interface ClusterSocketHistory {
  clusterId: string;
  socketId: string;
  sentPreflightUrls: string[];
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
      sentPreflightUrls: [],
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

  async preflightLoop() {
    if (!this.clusterSocketHistory) {
      return;
    }

    for (const clusterSocketHistory of this.clusterSocketHistory) {
      const apps = await this.kotsAppStore.listAppsForCluster(clusterSocketHistory.clusterId);
      for (const app of apps) {
        const pendingPreflightURLs = await this.preflightStore.getPendingPreflightUrls();
        for (const pendingPreflightURL of pendingPreflightURLs) {
          if (clusterSocketHistory.sentPreflightUrls.indexOf(pendingPreflightURL) === -1) {
            this.io.in(clusterSocketHistory.clusterId).emit("preflight", {uri: pendingPreflightURL});
            clusterSocketHistory.sentPreflightUrls.push(pendingPreflightURL);
          }
        }

        const deployedAppVersion = await this.kotsAppStore.getCurrentVersion(app.id, clusterSocketHistory.clusterId);
        const deployedAppSequence = deployedAppVersion && deployedAppVersion.sequence;
        if (deployedAppSequence! > -1) {
          if (clusterSocketHistory.sentDeploySequences.indexOf(`${app.id}/${deployedAppSequence!}`) === -1) {
            const desiredNamespace = ".";

            const cluster = await this.clusterStore.getCluster(clusterSocketHistory.clusterId);

            const rendered = await app.render(''+app.currentSequence, `overlays/downstreams/${cluster.title}`);
            const b = new Buffer(rendered);

            const args = {
              app_id: app.id,
              namespace: desiredNamespace,
              manifests: b.toString("base64"),
            }
            this.io.in(clusterSocketHistory.clusterId).emit("deploy", args);
            clusterSocketHistory.sentDeploySequences.push(`${app.id}/${deployedAppSequence!}`);

            try {
              const kotsAppSpec = await app.getKotsAppSpec(cluster.id, this.kotsAppStore)
              if (kotsAppSpec && kotsAppSpec.spec && kotsAppSpec.spec.statusInformers) {
                this.io.in(clusterSocketHistory.clusterId).emit("appInformers", {
                  app_id: app.id,
                  informers: kotsAppSpec.spec.statusInformers,
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
