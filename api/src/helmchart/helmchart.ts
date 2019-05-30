export interface HelmChart {
  id: string;
  clusterId: string;
  helmName: string
  namespace: string;
  version: number;
  firstDeployedAt: Date;
  lastDeployedAt: Date;
  isDeleted: boolean;
  chartVersion: string;
  appVersion: string;
}
