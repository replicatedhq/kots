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

  // Watch Cluster Methods
  public async getCluster(watchId: string, stores: Stores) {
    return stores.clusterStore.getForWatch(watchId!)
  }

  // Parent/Child Watch Methods
  public async getParentWatch(watchId: string) {}
  public async getChildWatches(stores): Promise<Watch[]> {
    return stores.watchStore.listWatches(undefined, this.id);
  }

  // Version Methods
  public async getCurrentVersion(watchId: string, stores: Stores) {
    return stores.watchStore.getCurrentVersion(watchId!);
  }
  public async getPendingVersions(watchId: string, stores: Stores) {
    return stores.watchStore.listPendingVersions(watchId!);
  }
  public async getPastVersions(watchId: string, stores: Stores) {
    return stores.watchStore.listPastVersions(watchId!);
  }

  // Contributor Methods
  public async getContributors(watchId: string, stores: Stores) {
    return stores.watchStore.listWatchContributors(watchId!);
  }

  public async addContributor() {}
  
  // Features Methods
  public async getFeatures(watchId: string, stores: Stores) {
    const features = await stores.featureStore.listWatchFeatures(watchId);
    const result = _.map(features, (feature: Feature) => {
      return {
        ...feature,
      };
    });
    return result;
  }

  public toSchema(watch: Watch, root: any, stores: Stores, context: Context): any {
    return {
      ...watch,
      watches: async () => (await watch.getChildWatches(stores)).map(childWatch => this.toSchema(childWatch!, root, stores, context)),
      cluster: async () => await this.getCluster(watch.id, stores),
      contributors: async () => this.getContributors(watch.id, stores),
      notifications: async () => NotificationQueries(stores).listNotifications(root, { watchId: watch.id! }, context),
      features: async () => this.getFeatures(watch.id!, stores),
      pendingVersions: async () => this.getPendingVersions(watch.id!, stores),
      pastVersions: async () => this.getPastVersions(watch.id!, stores),
      currentVersion: async () => this.getCurrentVersion(watch.id!, stores),
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
