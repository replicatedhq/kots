module.exports = {
  ENVIRONMENT: "staging",
  INSTALL_ENDPOINT: "https://www.staging.replicated.com/api/install",
  GRAPHQL_ENDPOINT: "https://www.staging.replicated.com/graphql",
  REST_ENDPOINT: "https://www.staging.replicated.com/api",
  SHIPDOWNLOAD_ENDPOINT: "https://www.staging.replicated.com/api/v1/download",
  SHIPINIT_ENDPOINT: "https://www.staging.replicated.com/api/v1/init/",
  SHIPUPDATE_ENDPOINT: "https://www.staging.replicated.com/api/v1/update/",
  SHIPEDIT_ENDPOINT: "https://www.staging.replicated.com/api/v1/edit/",
  GITHUB_CLIENT_ID: "Iv1.b16ae32898661e1d",
  GITHUB_REDIRECT_URI: "https://www.staging.replicated.com/auth/github/callback",
  GITHUB_INSTALL_URL: "https://github.com/apps/ship-cloud-staging",
  BUGSNAG_API_KEY: "a7504e4a1632ad844b789721e8982b19",
  SHOW_SCM_LEADS: true,
  AVALIABLE_LOGIN_TYPES: ["github"],
  WEBPACK_SCRIPTS: [
    "https://unpkg.com/react@16/umd/react.production.min.js",
    "https://unpkg.com/react-dom@16/umd/react-dom.production.min.js"
  ],
};
