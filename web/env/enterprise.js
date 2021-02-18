module.exports = {
  ENVIRONMENT: "production",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "/api/v1",
  KOTSADM_BUILD_VERSION: (function () {
    return process.env.KOTSADM_BUILD_VERSION;
  }()),
};

