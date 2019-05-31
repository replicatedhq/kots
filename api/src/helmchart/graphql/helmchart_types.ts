const HelmChart = `
  type HelmChart {
    id: ID
    clusterID: String
    helmName: String
    namespace: String
    version: Int
    firstDeployedAt: String
    lastDeployedAt: String
    isDeleted: Boolean
    chartVersion: String
    appVersion: String
  }
`;

export default [
  HelmChart,
];
