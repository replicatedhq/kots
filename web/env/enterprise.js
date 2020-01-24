module.exports = {
  ENVIRONMENT: "production",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "http://localhost:8800/api/v1",
  GRAPHQL_ENDPOINT: "http://localhost:8800/graphql",
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return process.env.SHIP_CLUSTER_BUILD_VERSION;
  }()),
};

