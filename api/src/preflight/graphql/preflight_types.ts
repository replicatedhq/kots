const PreflightResult = `
  type PreflightResult {
    watchId: String!
    result: String!
    createdAt: String!
  }
`;

const KotsPreflightResult = `
  type KotsPreflightResult {
    appId: String
    result: String
    updatedAt: String
    clusterId: String
  }
`;

export default [ PreflightResult, KotsPreflightResult ];
