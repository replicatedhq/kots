import { getApplicationType } from "@src/utilities/utilities";

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
    tabName: "downstreams",
    displayName: "Downstreams",
    to: slug => `/watch/${slug}/downstreams`,
    displayRule: watch => {
      return !watch.cluster;
    }
  },
  {
    tabName: "config",
    displayName: "Config",
    to: slug => `/watch/${slug}/config`,
    displayRule: watch => {
      return getApplicationType(watch) === "replicated.app";
    }
  },
  {
    tabName: "troubleshoot",
    displayName: "Troubleshoot",
    to: slug => `/watch/${slug}/troubleshoot`,
    displayRule: watch => {
      return getApplicationType(watch) === "replicated.app";
    }
  },
  {
    tabName: "license",
    displayName: "License",
    to: slug => `/watch/${slug}/license`,
    displayRule: watch => {
      return getApplicationType(watch) === "replicated.app";
    }
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
