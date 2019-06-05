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
    tabName: "version-history",
    displayName: "Version history",
    to: slug => `/watch/${slug}/version-history`
  },
  {
    tabName: "deployment-clusters",
    displayName: "Clusters",
    to: slug => `/watch/${slug}/deployment-clusters`,
    displayRule: watch => {
      return !watch.cluster;
    }
  },
  {
    tabName: "config",
    displayName: "Config",
    to: slug => `/watch/${slug}/config`
  },
  {
    tabName: "troubleshoot",
    displayName: "Troubleshoot",
    to: slug => `/watch/${slug}/troubleshoot`,
  },
  {
    tabName: "license",
    displayName: "License",
    to: slug => `/watch/${slug}/license`,
  }
  // {
  //   tabName: "integrations",
  //   displayName: "Integrations",
  //   to: slug => `/watch/${slug}/integrations`
  // },
  // {
  //   tabName: "state",
  //   displayName: "State JSON",
  //   to: slug => `/watch/${slug}/state`
  // }
];
