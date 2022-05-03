module.exports = {
  ENVIRONMENT: "development",
  API_ENDPOINT: `https://kotsadm-x-${process.env.OKTETO_NAMESPACE}-x-vcluster-${process.env.OKTETO_NAMESPACE}.replicated.okteto.dev/api/v1`,
  KOTSADM_BUILD_VERSION: Date.now()
};
