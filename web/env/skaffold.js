module.exports = {
  ENVIRONMENT: "development",
  INSTALL_ENDPOINT: "http://localhost:8065/api/install",
  GRAPHQL_ENDPOINT: "http://localhost:8065/graphql",
  REST_ENDPOINT: "http://localhost:8065/api",
  SHIPINIT_ENDPOINT: "http://localhost:8065/api/v1/init/",
  SHIPUPDATE_ENDPOINT: "http://localhost:8065/api/v1/update/",
  SHOW_SCM_LEADS: false,
  GITHUB_REDIRECT_URI: "http://localhost:8000/auth/github/callback",
  AVALIABLE_LOGIN_TYPES: ["github", "traditional"],
  WEBPACK_SCRIPTS: [
    "https://unpkg.com/react@16/umd/react.development.js",
    "https://unpkg.com/react-dom@16/umd/react-dom.development.js"
  ],
};
