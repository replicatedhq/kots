import {
  getApplicationType,
  isHelmChart
} from "@src/utilities/utilities";

export default [
  {
    tabName: "app",
    displayName: "Application",
    to: slug => `/watch/${slug}`,
    displayRule: watch => {
      return isHelmChart(watch) || !watch.cluster;
    }
  },
  {
    tabName: "version-history",
    displayName: "Version history",
    to: slug => `/watch/${slug}/version-history`,
    hasBadge: watch => {
      let downstreamPendingLengths = [];
      watch.watches?.map((w) => {
        downstreamPendingLengths.push(w.pendingVersions.length);
      });
      return downstreamPendingLengths.length > 0;
    },
    displayRule: watch => {
      return !isHelmChart(watch);
    },
  },
  {
    tabName: "downstreams",
    displayName: "Downstreams",
    to: slug => `/watch/${slug}/downstreams`,
    displayRule: watch => {
      if (isHelmChart(watch)) {
        return false;
      }
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
  },
  {
    tabName: "state",
    displayName: "State JSON",
    to: slug => `/watch/${slug}/state`,
    displayRule: watch => {
      if (isHelmChart(watch)) {
        return false;
      }
      return Boolean(watch.cluster) || getApplicationType(watch) !== "replicated.app";
    }
  }
  // {
  //   tabName: "integrations",
  //   displayName: "Integrations",
  //   to: slug => `/watch/${slug}/integrations`
  // }
];
