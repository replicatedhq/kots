module.exports = {
  ENVIRONMENT: "production",
  INSTALL_ENDPOINT: "###_INSTALL_ENDPOINT_###",
  GRAPHQL_ENDPOINT: "###_GRAPHQL_ENDPOINT_###",
  REST_ENDPOINT: "###_REST_ENDPOINT_###",
  GITHUB_CLIENT_ID: "###_GITHUB_CLIENT_ID_###",
  SHIPDOWNLOAD_ENDPOINT: "###_SHIPDOWNLOAD_ENDPOINT_###",
  SHIPINIT_ENDPOINT: "###_SHIPINIT_ENDPOINT_###",
  SHIPUPDATE_ENDPOINT: "###_SHIPUPDATE_ENDPOINT_###",
  SHIPEDIT_ENDPOINT: "###_SHIPEDIT_ENDPOINT_###",
  GITHUB_REDIRECT_URI: "###_GITHUB_REDIRECT_URI_###",
  GITHUB_INSTALL_URL: "###_GITHUB_INSTALL_URL_###",
  SHOW_SCM_LEADS: false,
  SECURE_ADMIN_CONSOLE: true,
  PROD_PERFECT_WRITE_KEY: "",
  AVAILABLE_LOGIN_TYPES: ["github", "traditional", "bitbucket", "gitlab"],
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return process.env.SHIP_CLUSTER_BUILD_VERSION;
  }()),
  WEBPACK_SCRIPTS: [
    "https://unpkg.com/react@16/umd/react.production.min.js",
    "https://unpkg.com/react-dom@16/umd/react-dom.production.min.js",
  ],
};
