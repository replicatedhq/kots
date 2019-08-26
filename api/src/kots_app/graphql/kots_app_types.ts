const KotsApp = `
  type KotsApp {
    id: String
    name: String
    iconUri: String
    createdAt: String
    updatedAt: String
    slug: String
    currentSequence: Int
    lastUpdateCheckAt: String
    downstreams: [KotsDownstream]
  }
`;

const KotsDownstream = `
  type KotsDownstream {
    name: String
    cluster: Cluster
  }
`;

export default [
  KotsApp,
  KotsDownstream,
];
