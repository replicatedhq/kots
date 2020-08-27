module.exports = {
  ENVIRONMENT: "production",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "/api/v1",
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return process.env.SHIP_CLUSTER_BUILD_VERSION;
  }()),
};

