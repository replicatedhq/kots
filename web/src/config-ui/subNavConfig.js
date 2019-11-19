import {
  getApplicationType,
  isHelmChart
} from "@src/utilities/utilities";

export default [
  {
    tabName: "app",
    displayName: "Application",
    to: (slug) => `/app/${slug}`,
    displayRule: watch => {
      return isHelmChart(watch) || !watch.cluster;
    }
  },
  {
    tabName: "version-history",
    displayName: "Version history",
    to: (slug) => `/app/${slug}/version-history`,
    hasBadge: watch => {
      let downstreamPendingLengths = [];
      watch.watches?.map((w) => {
        downstreamPendingLengths.push(w.pendingVersions.length);
      });
      return Math.max(...downstreamPendingLengths) > 0;
    },
    displayRule: watch => {
      return !isHelmChart(watch);
    },
  },
  {
    tabName: "config",
    displayName: "Config",
    to: (slug) => `/app/${slug}/config`,
    displayRule: watch => {
      return watch.isConfigurable || getApplicationType(watch) === "replicated.app";
    }
  },
  {
    tabName: "troubleshoot",
    displayName: "Troubleshoot",
    to: (slug) => `/app/${slug}/troubleshoot`,
  },
  {
    tabName: "license",
    displayName: "License",
    to: (slug) => `/app/${slug}/license`,
    displayRule: watch => {
      return watch?.upstreamUri?.startsWith("replicated://") || getApplicationType(watch) === "replicated.app";
    }
  },
  {
    tabName: "state",
    displayName: "State JSON",
    to: (slug) => `/app/${slug}/state`,
    displayRule: watch => {
      if (isHelmChart(watch) || watch.name) {
        return false;
      }
      return Boolean(!watch.cluster);
    }
  },
  {
    tabName: "tree",
    displayName: "View files",
    to: (slug, sequence) => `/app/${slug}/tree/${sequence}`,
    displayRule: watch => {
      return Boolean(watch.name);
    }
  },
  {
    tabName: "gitops",
    displayName: "GitOps",
    to: (slug) => `/app/${slug}/gitops`,
    displayRule: (watch) => {
      return watch?.downstreams && watch?.downstreams.length > 0 && watch.downstreams[0].gitops?.enabled;
    }
  },
  {
    tabName: "registry-settings",
    displayName: "Registry settings",
    to: (slug) => `/app/${slug}/registry-settings`,
    displayRule: () => {
      return true;
    }
  },
  // {
  //   tabName: "integrations",
  //   displayName: "Integrations",
  //   to: (slug) => `/app/${slug}/integrations`
  // }
];
