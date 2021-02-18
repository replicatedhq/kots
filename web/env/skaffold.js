module.exports = {
  ENVIRONMENT: "development",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "/api/v1",
  KOTSADM_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
