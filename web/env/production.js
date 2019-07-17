module.exports = {
  ENVIRONMENT: "production",
  INSTALL_ENDPOINT: "https://www.replicated.com/api/install",
  GRAPHQL_ENDPOINT: "https://www.replicated.com/graphql",
  REST_ENDPOINT: "https://www.replicated.com/api",
  GITHUB_CLIENT_ID: "Iv1.31f2356a7342e85f",
  SHIPDOWNLOAD_ENDPOINT: "https://www.replicated.com/api/v1/download",
  SHIPINIT_ENDPOINT: "https://www.replicated.com/api/v1/init/",
  SHIPUPDATE_ENDPOINT: "https://www.replicated.com/api/v1/update/",
  SHIPEDIT_ENDPOINT: "https://www.replicated.com/api/v1/edit/",
  GITHUB_REDIRECT_URI: "https://www.replicated.com/auth/github/callback",
  GITHUB_INSTALL_URL: "https://github.com/apps/replicated-ship/installations/new",
  BUGSNAG_API_KEY: "a7504e4a1632ad844b789721e8982b19",
  PROD_PERFECT_WRITE_KEY: "VDFMDV5Z2FVHU9L20S58LHPX69Z0ZQQ0ZXXIRKHTI37MY1MTSQ8KLFB01QCKEIHY57AQBKPVD9O2VUFKNOV8BA8ZBSBPWVD460ORHLDVPPBFKAKUH2W3WFLJQF1JERKM16LXG1Q4D12JDT3ZIX6PZ51O2UQMTEUXIVG1MX6I3LVC5HDMBSAGJBAD9CUQQA5L",
  PUBLIC_ASSET_PATH: "*https://www.replicated.com/",
  SHOW_SCM_LEADS: true,
  SHIP_CLUSTER_BUILD_VERSION: (function () {
    return process.env.SHIP_CLUSTER_BUILD_VERSION;
  }()),
  AVAILABLE_LOGIN_TYPES: ["github"],
  WEBPACK_SCRIPTS: [
    "https://unpkg.com/react@16/umd/react.production.min.js",
    "https://unpkg.com/react-dom@16/umd/react-dom.production.min.js",
    {
      src: "/prodPerfect.js",
      type: "text/javascript",
    },
  ],
};
