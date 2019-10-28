import {
  getApplicationType,
  isHelmChart
} from "@src/utilities/utilities";

export default [
  {
    tabName: "app",
    displayName: "Application",
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}`,
    displayRule: watch => {
      return isHelmChart(watch) || !watch.cluster;
    }
  },
  {
    tabName: "version-history",
    displayName: "Version history",
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/version-history`,
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
    tabName: "downstreams",
    displayName: "Downstreams",
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/downstreams`,
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
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/config`,
    displayRule: watch => {
      return watch.isConfigurable || getApplicationType(watch) === "replicated.app";
    }
  },
  {
    tabName: "troubleshoot",
    displayName: "Troubleshoot",
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/troubleshoot`,
  },
  {
    tabName: "license",
    displayName: "License",
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/license`,
    displayRule: watch => {
      return watch?.upstreamUri?.startsWith("replicated://") || getApplicationType(watch) === "replicated.app";
    }
  },
  {
    tabName: "state",
    displayName: "State JSON",
    to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/state`,
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
    to: (slug, isKots, sequence) => `/app/${slug}/tree/${isKots ? sequence : ""}`,
    displayRule: watch => {
      return Boolean(watch.name);
    }
  },
  {
    tabName: "airgap-settings",
    displayName: "Airgap settings",
    to: (slug) => `/app/${slug}/airgap-settings`,
    displayRule: watch => {
      // watch.isAirgap is already typed as a Boolean from GraphQL
      return watch.isAirgap;
    }
  },
  // {
  //   tabName: "integrations",
  //   displayName: "Integrations",
  //   to: (slug, isKots) => `/${isKots ? "app" : "watch"}/${slug}/integrations`
  // }
];
