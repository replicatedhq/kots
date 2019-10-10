import { IO, Nsp, SocketService, SocketSession, Socket } from "@tsed/socketio";
import { getPostgresPool } from "../../util/persistence/db";
import { KotsAppStore } from "../../kots_app/kots_app_store";
import { Params } from "../../server/params";
import { ClusterStore } from "../../cluster";
import { PreflightStore } from "../../preflight/preflight_store";
import _ from "lodash";

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
  clusterStore: ClusterStore;
  preflightStore: PreflightStore;
  clusterSocketHistory: ClusterSocketHistory[];

  constructor(@IO private io: SocketIO.Server) {
    getPostgresPool()
      .then((pool) => {
        Params.getParams()
          .then((params) => {
            this.kotsAppStore = new KotsAppStore(pool, params);
            this.clusterStore = new ClusterStore(pool, params);
            this.preflightStore = new PreflightStore(pool);
            this.clusterSocketHistory = [];

            setInterval(this.preflightLoop.bind(this), 1000);
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
              "app_id": app.id,
              namespace: desiredNamespace,
              manifests: b.toString("base64"),
            }
            this.io.in(clusterSocketHistory.clusterId).emit("deploy", args);
            clusterSocketHistory.sentDeploySequences.push(`${app.id}/${deployedAppSequence!}`);
          }
        }
      }
    }
  }
}
