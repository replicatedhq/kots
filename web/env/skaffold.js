module.exports = {
  ENVIRONMENT: "development",
  INSTALL_ENDPOINT: "http://localhost:30065/api/install",
  GRAPHQL_ENDPOINT: "http://localhost:30065/graphql",
  REST_ENDPOINT: "http://localhost:30065/api",
  SHIPDOWNLOAD_ENDPOINT: "http://localhost:30065/api/v1/download",
  SHIPINIT_ENDPOINT: "http://localhost:30065/api/v1/init/",
  SHIPUPDATE_ENDPOINT: "http://localhost:30065/api/v1/update/",
  SHIPEDIT_ENDPOINT: "http://localhost:30065/api/v1/edit/",
  SHOW_SCM_LEADS: false,
  GITHUB_REDIRECT_URI: "http://localhost:30065/auth/github/callback",
  SECURE_ADMIN_CONSOLE: true,
  DISABLE_KOTS: false,
  AVAILABLE_LOGIN_TYPES: [],
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
