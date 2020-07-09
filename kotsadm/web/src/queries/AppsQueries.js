import gql from "graphql-tag";

export const ping = gql(`
  query ping {
    ping
  }
`);

export const listAppsRaw = `
  query listApps {
    listApps {
      kotsApps {
        id
        name
        iconUri
        createdAt
        updatedAt
        slug
        currentSequence
        isGitOpsSupported
        allowSnapshots
        licenseType
        updateCheckerSpec
        currentVersion {
          title
          status
          createdOn
          sequence
          deployedAt
          yamlErrors {
            path
            error
          }
        }
        lastUpdateCheckAt
        downstreams {
          name
          currentVersion {
            title
            status
            createdOn
            sequence
            deployedAt
            yamlErrors {
              path
              error
            }
          }
          gitops {
            enabled
            provider
            uri
            hostname
            path
            branch
            format
            action
            isConnected
          }
          pendingVersions {
            title
            status
            createdOn
            sequence
            deployedAt
            yamlErrors {
              path
              error
            }
          }
          pastVersions {
            title
            status
            createdOn
            sequence
            deployedAt
            yamlErrors {
              path
              error
            }
          }
          cluster {
            id
            title
            slug
            createdOn
            lastUpdated
            currentVersion {
              title
              status
              createdOn
              sequence
              deployedAt
              yamlErrors {
                path
                error
              }
            }
            shipOpsRef {
              token
            }
            totalApplicationCount
          }
        }
      }
    }
  }
`;
export const listApps = gql(listAppsRaw);

export const getKotsAppRaw = `
  query getKotsApp($slug: String!) {
    getKotsApp(slug: $slug) {
      id
      name
      iconUri
      createdAt
      updatedAt
      slug
      upstreamUri
      currentSequence
      hasPreflight
      isAirgap
      isConfigurable
      isGitOpsSupported
      allowRollback
      allowSnapshots
      updateCheckerSpec
      currentVersion {
        title
        status
        createdOn
        sequence
        releaseNotes
        deployedAt
        yamlErrors {
          path
          error
        }
      }
      lastUpdateCheckAt
      bundleCommand
      downstreams {
        name
        links {
          title
          uri
        }
        currentVersion {
          title
          status
          createdOn
          sequence
          deployedAt
          source
          releaseNotes
          parentSequence
          yamlErrors {
            path
            error
          }
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          deployedAt
          parentSequence
          yamlErrors {
            path
            error
          }
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          deployedAt
          parentSequence
          yamlErrors {
            path
            error
          }
        }
        gitops {
          enabled
          provider
          uri
          hostname
          path
          branch
          format
          action
          deployKey
          isConnected
        }
        cluster {
          id
          title
          slug
          createdOn
          lastUpdated
          currentVersion {
            title
            status
            createdOn
            sequence
            deployedAt
            yamlErrors {
              path
              error
            }
          }
          shipOpsRef {
            token
          }
          totalApplicationCount
        }
      }
    }
  }
`;
export const getKotsApp = gql(getKotsAppRaw);

export const listDownstreamsForAppRaw = `
  query listDownstreamsForApp($slug: String!) {
    listDownstreamsForApp(slug: $slug) {
      id
      title
      slug
      createdOn
      lastUpdated
      currentVersion {
        title
        status
        createdOn
        sequence
        deployedAt
        yamlErrors {
          path
          error
        }
      }
      shipOpsRef {
        token
      }
      totalApplicationCount
    }
  }
`;

export const listDownstreamsForApp = gql(listDownstreamsForAppRaw);

export const getKotsDownstreamHistoryRaw = `
  query getKotsDownstreamHistory($clusterSlug: String!, $upstreamSlug: String!) {
    getKotsDownstreamHistory(clusterSlug: $clusterSlug, upstreamSlug: $upstreamSlug) {
      title
      status
      createdOn
      sequence
      parentSequence
      releaseNotes
      deployedAt
      source
      diffSummary
      preflightResult
      preflightResultCreatedAt
      commitUrl
      gitDeployable
    }
  }
`;
export const getKotsDownstreamHistory = gql(getKotsDownstreamHistoryRaw);

export const getKotsPreflightResultRaw = `
  query getKotsPreflightResult($appSlug: String!, $clusterSlug: String!, $sequence: Int!) {
    getKotsPreflightResult(appSlug: $appSlug, clusterSlug: $clusterSlug, sequence: $sequence) {
      appSlug
      clusterSlug
      result
      createdAt
    }
  }
`;
export const getKotsPreflightResult = gql(getKotsPreflightResultRaw);

export const getLatestKotsPreflightResultRaw = `
  query getLatestKotsPreflightResult {
    getLatestKotsPreflightResult {
      appSlug
      clusterSlug
      result
      createdAt
    }
  }
`;
export const getLatestKotsPreflightResult = gql(getLatestKotsPreflightResultRaw);

export const getAirgapInstallStatusRaw = `
  query getAirgapInstallStatus {
    getAirgapInstallStatus {
      installStatus
      currentMessage
    }
  }
`;
export const getAirgapInstallStatus = gql(getAirgapInstallStatusRaw);

export const getOnlineInstallStatusRaw = `
  query getOnlineInstallStatus {
    getOnlineInstallStatus {
      installStatus
      currentMessage
    }
  }
`;
export const getOnlineInstallStatus = gql(getOnlineInstallStatusRaw);

export const getImageRewriteStatusRaw = `
  query getImageRewriteStatus {
    getImageRewriteStatus {
      currentMessage
      status
    }
  }
`;
export const getImageRewriteStatus = gql(getImageRewriteStatusRaw);

export const getUpdateDownloadStatusRaw = `
  query getUpdateDownloadStatus {
    getUpdateDownloadStatus {
      currentMessage
      status
    }
  }
`;
export const getUpdateDownloadStatus = gql(getUpdateDownloadStatusRaw);

export const getAppConfigGroups = gql`
  query getAppConfigGroups($slug: String!, $sequence: Int!) {
    getAppConfigGroups(slug: $slug, sequence: $sequence) {
      name
      title
      description
      items {
        name
        type
        title
        help_text
        recommended
        default
        value
        error
        data
        multi_value
        readonly
        write_once
        when
        multiple
        hidden
        position
        affix
        required
        items {
          name
          title
          recommended
          default
          value
        }
      }
    }
  }
`;

export const getKotsDownstreamOutput = gql`
  query getKotsDownstreamOutput($appSlug: String!, $clusterSlug: String!, $sequence: Int!) {
    getKotsDownstreamOutput(appSlug: $appSlug, clusterSlug: $clusterSlug, sequence: $sequence) {
      dryrunStdout
      dryrunStderr
      applyStdout
      applyStderr
      renderError
    }
  }
`;

export const getAppLicense = gql`
  query getAppLicense($appId: String!) {
    getAppLicense(appId: $appId) {
      id
      expiresAt
      channelName
      licenseSequence
      licenseType
      entitlements {
        title
        value
        label
      }
    }
  }
`;

export const templateConfigGroups = gql`
  query templateConfigGroups($slug: String!, $sequence: Int!, $configGroups: [KotsConfigGroupInput]!) {
    templateConfigGroups(slug: $slug, sequence: $sequence, configGroups: $configGroups) {
      name
      title
      description
      items {
        name
        type
        title
        help_text
        recommended
        default
        value
        error
        data
        multi_value
        readonly
        write_once
        when
        multiple
        hidden
        position
        affix
        required
        items {
          name
          title
          recommended
          default
        }
      }
    }
  }
`;

export const getKotsAppDashboard = gql`
  query getKotsAppDashboard($slug: String!, $clusterId: String) {
    getKotsAppDashboard(slug: $slug, clusterId: $clusterId) {
      appStatus {
        appId
        updatedAt
        state
        resourceStates {
          kind
          name
          namespace
          state
        }
      }
      metrics {
        title
        tickFormat
        tickTemplate
        series {
          legendTemplate
          metric {
            name
            value
          }
          data {
            timestamp
            value
          }
        }
      }
      prometheusAddress
    }
  }
`;

export const getPrometheusAddress = gql`
  query getPrometheusAddress {
    getPrometheusAddress
  }
`;

export const getGitOpsRepo = gql`
  query getGitOpsRepo {
    getGitOpsRepo {
      enabled
      uri
      provider
      hostname
    }
  }
`;

export const getPreflightCommandRaw = `
  query getPreflightCommand($appSlug: String, $clusterSlug: String, $sequence: String) {
    getPreflightCommand(appSlug: $appSlug, clusterSlug: $clusterSlug, sequence: $sequence)
  }
`;

export const getPreflightCommand = gql(getPreflightCommandRaw);
