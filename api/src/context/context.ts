import { SessionStore, Session } from "../session";
import { getPostgresPool } from "../util/persistence/db";
import { Params } from "../server/params";
import { ReplicatedError } from "../server/errors";
import { isAfter } from "date-fns";
import { Cluster } from "../cluster";
import _ from "lodash";
import { Stores } from "../schema/stores";
import { Watch } from "../watch";
import slugify from "slugify";
import { Notification } from "../generated/types";
import { KotsApp } from "../kots_app";

export class Context {
  constructor(private readonly stores: Stores) {}

  public session: Session;

  public static async fetch(stores: Stores, token: string): Promise<Context> {
    const pool = await getPostgresPool();
    const params = await Params.getParams();
    const sessionStore = new SessionStore(pool, params);

    const context = new Context(stores);
    context.session = await sessionStore.decode(token);
    return context;
  }

  public sessionType(): string {
    return this.session.type;
  }

  public getGitHubToken(): string {
    return this.session.scmToken;
  }

  public requireSingleTenantSession() {
    if (!this.session) {
      throw new ReplicatedError("Unauthorized");
    }

    return true;
  }

  public hasValidSession(): ReplicatedError | null {
    if (this.getGitHubToken().length === 0) {
      return new ReplicatedError("Unauthorized");
    }

    const currentTime = new Date(Date.now()).toUTCString();
    if (isAfter(currentTime, this.session.expiresAt)) {
      return new ReplicatedError("Expired session");
    }

    return null;
  }

  public async getUsername(): Promise<string> {
    const shipUser = await this.stores.userStore.getUser(this.session.userId);
    if (shipUser.githubUser) {
      return shipUser.githubUser.login;
    } else if (shipUser.shipUser) {
      return slugify(`${shipUser.shipUser.firstName}-${shipUser.shipUser.lastName}`).toLowerCase();
    }

    throw new ReplicatedError("unable to get username");
  }

  public async listClusters(): Promise<Cluster[]> {
    const userClusters = await this.stores.clusterStore.listClusters(this.session.userId);
    const allUserClusters = await this.stores.clusterStore.listAllUsersClusters();

    return [
      ...userClusters,
      ...allUserClusters,
    ];
  }

  public async getCluster(id: string): Promise<Cluster> {
    const clusters = await this.stores.clusterStore.listClusters(this.session.userId);
    const cluster = _.find(clusters, (cluster: Cluster) => {
      return cluster.id === id;
    });

    if (!cluster) {
      throw new ReplicatedError("Cluster not found");
    }

    return cluster;
  }

  public async findWatch(slug: string): Promise<Watch> {
    const watch = await this.stores.watchStore.findUserWatch(this.session.userId, { slug });

    if (!watch) {
      throw new ReplicatedError("Watch not found");
    }

    return watch;
  }

  public async getWatch(id: string): Promise<Watch> {
    const watch = await this.stores.watchStore.findUserWatch(this.session.userId, { id });

    if (!watch) {
      throw new ReplicatedError("Watch not found");
    }

    return watch;
  }

  public async getApp(id: string): Promise<KotsApp> {
    const app = await this.stores.kotsAppStore.getApp(id);

    if (!app) {
      throw new ReplicatedError("App not found");
    }

    return app;
  }

  public async getNotification(id: string): Promise<Notification> {
    const notification = await this.stores.notificationStore.findUserNotification(this.session.userId, id);

    if (!notification) {
      throw new ReplicatedError("Notification not found");
    }

    return notification;
  }
}
