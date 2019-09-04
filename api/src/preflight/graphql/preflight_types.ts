const PreflightResult = `
  type PreflightResult {
    watchId: String!
    result: String!
    createdAt: String!
  }
`;

const KotsPreflightResult = `
  type KotsPreflightResult {
    result: String
    updatedAt: String
  }
`;

export default [ PreflightResult, KotsPreflightResult ];
