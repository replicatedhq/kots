import {
  getApplicationType,
  isHelmChart,
  Utilities,
} from "@src/utilities/utilities";

import { rbacRoles } from "@src/constants/rbac";

export default [
  {
    tabName: "app",
    displayName: "Dashboard",
    to: (slug) => `/app/${slug}`,
    displayRule: ({ app }) => {
      return isHelmChart(app) || !app.cluster;
    }
  },
  {
    tabName: "version-history",
    displayName: "Version history",
    to: (slug) => `/app/${slug}/version-history`,
    hasBadge: ({ app }) => {
      let downstreamPendingLengths = [];
      app.watches?.map((w) => {
        downstreamPendingLengths.push(w.pendingVersions.length);
      });
      return Math.max(...downstreamPendingLengths) > 0;
    },
    displayRule: ({ app }) => {
      return !isHelmChart(app);
    },
  },
  {
    tabName: "config",
    displayName: "Config",
    to: (slug, sequence, configSequence) => `/app/${slug}/config/${configSequence}`,
    displayRule: ({ app }) => {
      return app.isConfigurable || getApplicationType(app) === "replicated.app";
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
    displayRule: ({ app }) => {
      return app?.upstreamUri?.startsWith("replicated://") || getApplicationType(app) === "replicated.app";
    }
  },
  {
    tabName: "state",
    displayName: "State JSON",
    to: (slug) => `/app/${slug}/state`,
    displayRule: ({ app }) => {
      if (isHelmChart(app) || app.name) {
        return false;
      }
      return Boolean(!app.cluster);
    }
  },
  {
    tabName: "tree",
    displayName: "View files",
    to: (slug, sequence) => `/app/${slug}/tree/${sequence}`,
    displayRule: ({ app, isHelmManaged }) => {
      return !isHelmManaged &&
        Boolean(app.name) &&
        Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]);
    }
  },
  {
    tabName: "gitops",
    displayName: "GitOps",
    to: (slug) => `/app/${slug}/gitops`,
    displayRule: ({ app }) => {
      return app.downstream?.gitops?.enabled;
    }
  },
  {
    tabName: "registry-settings",
    displayName: "Registry settings",
    to: (slug) => `/app/${slug}/registry-settings`,
    displayRule: ({ isHelmManaged })=> {
      console.log(isHelmManaged)
      return !isHelmManaged;
    },
  },
  {
    tabName: "access",
    displayName: "Access",
    to: (slug) => `/app/${slug}/access`,
    displayRule: ({ isIdentityServiceSupported }) => {
      return isIdentityServiceSupported;
    }
  }
];
