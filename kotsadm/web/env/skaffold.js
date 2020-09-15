module.exports = {
  ENVIRONMENT: "development",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "http://127.0.0.1:30065/api/v1",
  KOTSADM_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
