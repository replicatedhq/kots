export default [
  {
    tabName: "app",
    displayName: "Application",
    to: slug => `/watch/${slug}`,
    displayRule: watch => {
      return !watch.cluster;
    }
  },
  {
    tabName: "deployment-clusters",
    displayName: "Deployment clusters",
    to: slug => `/watch/${slug}/deployment-clusters`,
    displayRule: watch => {
      return !watch.cluster;
    }
  },
  {
    tabName: "integrations",
    displayName: "Integrations",
    to: slug => `/watch/${slug}/integrations`
  },
  {
    tabName: "state",
    displayName: "State JSON",
    to: slug => `/watch/${slug}/state`
  }
];
