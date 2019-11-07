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
  SECURE_ADMIN_CONSOLE: false,
  DISABLE_KOTS: false,
  AVAILABLE_LOGIN_TYPES: ["github", "traditional"],
  NO_APPS_REDIRECT: "/upload-license",
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return String(Date.now());
  }()),
};
