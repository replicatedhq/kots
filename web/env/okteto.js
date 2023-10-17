module.exports = {
  ENVIRONMENT: "development",
  API_ENDPOINT: `https://kotsadm-${process.env.OKTETO_NAMESPACE}.okteto.repldev.com/api/v1`,
  KOTSADM_BUILD_VERSION: Date.now(),
};
