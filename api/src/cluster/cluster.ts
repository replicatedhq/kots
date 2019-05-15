
export class Cluster {
  id: string;
  title: string;
  slug: string;
  lastUpdated?: Date;
  createdOn: Date;
  gitOpsRef?: GitOpsRef;
  shipOpsRef?: ShipOpsRef;
}

export interface GitOpsRef {
  owner: string;
  repo: string;
  branch: string;
  path: string;
}

export interface ShipOpsRef {
  token: string;
}
