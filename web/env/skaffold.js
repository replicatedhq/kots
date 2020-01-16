module.exports = {
  ENVIRONMENT: "development",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "http://localhost:30065/api/v1",
  GRAPHQL_ENDPOINT: "http://localhost:30065/graphql",
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
