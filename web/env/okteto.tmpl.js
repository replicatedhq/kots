module.exports = {
  ENVIRONMENT: "development",
  SECURE_ADMIN_CONSOLE: true,
  API_ENDPOINT: "https://kotsadm-$OKTETO_NAMESPACE.replicated.okteto.dev/api/v1",
  KOTSADM_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
