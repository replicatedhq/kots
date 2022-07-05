module.exports = {
  ENVIRONMENT: "development",
  API_ENDPOINT: `https://kotsadm-${process.env.OKTETO_NAMESPACE}.replicated.repldev.com/api/v1`,
  KOTSADM_BUILD_VERSION: Date.now()
};
