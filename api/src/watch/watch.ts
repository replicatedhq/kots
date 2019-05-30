import { Cluster } from "../cluster/cluster";
import { Feature } from "../feature/feature";
import { Stores } from "../schema/stores";
import { NotificationQueries } from "../notification";
import { Context } from "../context";
import * as _ from "lodash";

export class Watch {
  public id: string;
  public stateJSON: string;
  public watchName: string;
  public slug: string;
  public watchIcon: string;
  public lastUpdated: string;
  public createdOn: string;
  public contributors: [Contributor];
  public notifications: [Notification];
  public features: [Feature];
  public cluster: Cluster;
  public watches: [Watch];
  public currentVersion: Version;
  public pendingVersions: [Version];
  public pastVersions: [Version];
  public parentWatch: Watch;
  public metadata: string;

  // Watch Cluster Methods
  public async getCluster(stores: Stores): Promise<Cluster | void> {
    return stores.clusterStore.getForWatch(this.id)
  }

  // Parent/Child Watch Methods
  public async getParentWatch(stores: Stores): Promise<Watch> {
    const parentWatchId = await stores.watchStore.getParentWatchId(this.id)
    return stores.watchStore.getWatch(parentWatchId);
  }
  public async getChildWatches(stores): Promise<Watch[]> {
    return stores.watchStore.listWatches(undefined, this.id);
  }

  // Version Methods
  public async getCurrentVersion(stores: Stores): Promise<Version | undefined> {
    return stores.watchStore.getCurrentVersion(this.id);
  }
  public async getPendingVersions(stores: Stores): Promise<Version[]> {
    return stores.watchStore.listPendingVersions(this.id);
  }
  public async getPastVersions(stores: Stores): Promise<Version[]> {
    return stores.watchStore.listPastVersions(this.id);
  }

  // Contributor Methods
  public async getContributors(stores: Stores): Promise<Contributor[]> {
    return stores.watchStore.listWatchContributors(this.id);
  }

  public async addContributor(stores: Stores, context: Context): Promise<Contributor[]> {
    // Remove existing contributors
    await stores.userStoreOld.removeExistingWatchContributorsExcept(this.id, context.session.userId);

    // For each contributor, get user, if !user then create a new user
    for (const contributor of this.contributors) {
      const { githubId, login, avatar_url } = contributor!;

      let shipUser = await stores.userStore.tryGetGitHubUser(githubId!);
      if (!shipUser) {
        shipUser = await stores.userStore.createGitHubUser(githubId!, login!, avatar_url!, "");

        const allUsersClusters = await stores.clusterStore.listAllUsersClusters();
        for (const allUserCluster of allUsersClusters) {
          await stores.clusterStore.addUserToCluster(allUserCluster.id!, shipUser[0].id);
        }
      }
      // tslint:disable-next-line:curly
      if (shipUser[0].id !== context.session.userId) await stores.userStoreOld.saveWatchContributor(shipUser[0].id, this.id);
    }
    return this.getContributors(stores);
  }

  // Features Methods
  public async getFeatures(stores: Stores): Promise<Feature[]> {
    const features = await stores.featureStore.listWatchFeatures(this.id);
    const result = _.map(features, (feature: Feature) => {
      return {
        ...feature,
      };
    });
    return result;
  }

  public toSchema(root: any, stores: Stores, context: Context): any {
    return {
      ...this,
      watches: async () => (await this.getChildWatches(stores)).map(watch => watch.toSchema(root, stores, context)),
      cluster: async () => await this.getCluster(stores),
      contributors: async () => this.getContributors(stores),
      notifications: async () => NotificationQueries(stores).listNotifications(root, { watchId: this.id }, context),
      features: async () => this.getFeatures(stores),
      pendingVersions: async () => this.getPendingVersions(stores),
      pastVersions: async () => this.getPastVersions(stores),
      currentVersion: async () => this.getCurrentVersion(stores),
      parentWatch: async () => this.getParentWatch(stores),
    };
  }

}

export interface Version {
  title: string;
  status: string;
  createdOn: string;
  sequence: number;
  pullrequestNumber: number;
}

export interface VersionDetail {
  title: string;
  status: string;
  createdOn: string;
  sequence: number;
  pullrequestNumber: number;
  rendered: string;
}

export interface StateMetadata {
  name: string;
  icon: string;
  version: string;
}

export interface Contributor {
  id: string;
  createdAt: string;
  githubId: number;
  login: string;
  avatar_url: string;
}
