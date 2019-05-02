module.exports = {
  ENVIRONMENT: "development",
  INSTALL_ENDPOINT: "http://localhost:8065/api/install",
  GRAPHQL_ENDPOINT: "http://localhost:8065/graphql",
  SHIPINIT_ENDPOINT: "http://localhost:8065/api/v1/init/",
  SHIPUPDATE_ENDPOINT: "http://localhost:8065/api/v1/update/",
  GITHUB_REDIRECT_URI: "http://localhost:8000/auth/github/callback",
  WEBPACK_SCRIPTS: [
    "https://unpkg.com/react@16/umd/react.development.js",
    "https://unpkg.com/react-dom@16/umd/react-dom.development.js"
  ],
};
