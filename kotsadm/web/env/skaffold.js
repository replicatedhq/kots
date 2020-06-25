module.exports = {
  ENVIRONMENT: "development",
  SECURE_ADMIN_CONSOLE: true,
  SUBPATH: "/test",
  API_ENDPOINT: "http://127.0.0.1:30065/api/v1",
  GRAPHQL_ENDPOINT: "http://127.0.0.1:30065/graphql",
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
