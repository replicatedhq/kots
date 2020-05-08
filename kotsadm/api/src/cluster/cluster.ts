import { Stores } from "../schema/stores";

export class Cluster {
  id: string;
  title: string;
  slug: string;
  lastUpdated?: Date;
  createdOn: Date;
  shipOpsRef?: ShipOpsRef;

  public async getCurrentVersionOnCluster(appId: string, stores: Stores) {
    return stores.kotsAppStore.getCurrentVersion(appId, this.id);
  }

  public toKotsAppSchema(appId: string, stores: Stores) {
    return {
      ...this,
      currentVersion: () => this.getCurrentVersionOnCluster(appId, stores)
    }
  }
}

export interface ShipOpsRef {
  token: string;
}
