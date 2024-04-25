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
    displayRule: ({ app }) => isHelmChart(app) || !app.cluster,
  },
  {
    tabName: "version-history",
    displayName: "Version history",
    to: (slug) => `/app/${slug}/version-history`,
    hasBadge: ({ app }) => {
      const downstreamPendingLengths = [];
      app.watches?.forEach((w) => {
        downstreamPendingLengths.push(w.pendingVersions.length);
      });
      return Math.max(...downstreamPendingLengths) > 0;
    },
    displayRule: ({ app }) => !isHelmChart(app),
  },
  {
    tabName: "config",
    displayName: "Config",
    to: (slug, sequence, configSequence) =>
      `/app/${slug}/config/${configSequence}`,
    displayRule: ({ app }) =>
      app.isConfigurable || getApplicationType(app) === "replicated.app",
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
    displayRule: ({ app }) =>
      app?.upstreamUri?.startsWith("replicated://") ||
      getApplicationType(app) === "replicated.app",
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
    },
  },
  {
    tabName: "tree",
    displayName: "View files",
    to: (slug, sequence) => `/app/${slug}/tree/${sequence}`,
    displayRule: ({ app }) =>
      Boolean(app.name) &&
      Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]),
  },
  {
    tabName: "registry-settings",
    displayName: "Registry settings",
    to: (slug) => `/app/${slug}/registry-settings`,
  },
  {
    tabName: "access",
    displayName: "Access",
    to: (slug) => `/app/${slug}/access`,
    displayRule: ({ isIdentityServiceSupported }) => isIdentityServiceSupported,
  },
];
