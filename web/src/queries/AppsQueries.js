import gql from "graphql-tag";

export const ping = gql(`
  query ping {
    ping
  }
`);

export const getKotsMetadataRaw = `
  query getKotsMetadata {
    getKotsMetadata {
      name
      iconUri
      namespace
      isKurlEnabled
    }
  }
`;

export const getKotsMetadata = gql(getKotsMetadataRaw);

export const listAppsRaw = `
  query listApps {
    listApps {
      watches {
        id
        stateJSON
        watchName
        slug
        watchIcon
        createdOn
        lastUpdated
        metadata
        lastUpdateCheck
        contributors {
          id
          createdAt
          githubId
          login
          avatar_url
        }
        currentVersion {
          title
          status
          createdOn
          sequence
          pullrequestNumber
          deployedAt
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
          deployedAt
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
          deployedAt
        }
        notifications {
          id
          createdOn
          updatedOn
          triggeredOn
          enabled
          webhook {
            uri
          }
          email {
            recipientAddress
          }
        }
        watches {
          id
          stateJSON
          watchName
          slug
          watchIcon
          createdOn
          lastUpdated
          metadata
          lastUpdateCheck
          contributors {
            id
            createdAt
            githubId
            login
            avatar_url
          }
          currentVersion {
            title
            status
            createdOn
            sequence
            pullrequestNumber
            deployedAt
          }
          pendingVersions {
            title
            status
            createdOn
            sequence
            pullrequestNumber
            deployedAt
          }
          pastVersions {
            title
            status
            createdOn
            sequence
            pullrequestNumber
            deployedAt
          }
          notifications {
            id
            createdOn
            updatedOn
            triggeredOn
            enabled
            webhook {
              uri
            }
            email {
              recipientAddress
            }
          }
          cluster {
            id
            title
            slug
            createdOn
            lastUpdated
            gitOpsRef {
              owner
              repo
              branch
              path
            }
            shipOpsRef {
              token
            }
          }
        }
      }
      pendingUnforks {
        id
        clusterId
        helmName
        namespace
        version
        firstDeployedAt
        lastDeployedAt
        isDeleted
        chartVersion
        appVersion
      }
      kotsApps {
        id
        name
        iconUri
        createdAt
        updatedAt
        slug
        currentSequence
        currentVersion {
          title
          status
          createdOn
          sequence
          deployedAt
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
          }
          gitops {
            enabled
            provider
            uri
            path
            branch
            format
          }
          pendingVersions {
            title
            status
            createdOn
            sequence
            deployedAt
          }
          pastVersions {
            title
            status
            createdOn
            sequence
            deployedAt
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
            }
            gitOpsRef {
              owner
              repo
              branch
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

export const listWatchesRaw = `
  query listApps {
    watches {
      id
      stateJSON
      watchName
      slug
      watchIcon
      createdOn
      lastUpdated
      metadata
      lastUpdateCheck
      contributors {
        id
        createdAt
        githubId
        login
        avatar_url
      }
      currentVersion {
        title
        status
        createdOn
        sequence
        pullrequestNumber
        deployedAt
      }
      pendingVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
        deployedAt
      }
      pastVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
        deployedAt
      }
      notifications {
        id
        createdOn
        updatedOn
        triggeredOn
        enabled
        webhook {
          uri
        }
        email {
          recipientAddress
        }
      }
      watches {
        id
        stateJSON
        watchName
        slug
        watchIcon
        createdOn
        lastUpdated
        metadata
        lastUpdateCheck
        contributors {
          id
          createdAt
          githubId
          login
          avatar_url
        }
        currentVersion {
          title
          status
          createdOn
          sequence
          pullrequestNumber
          deployedAt
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
          deployedAt
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
          deployedAt
        }
        notifications {
          id
          createdOn
          updatedOn
          triggeredOn
          enabled
          webhook {
            uri
          }
          email {
            recipientAddress
          }
        }
        cluster {
          id
          title
          slug
          createdOn
          lastUpdated
          gitOpsRef {
            owner
            repo
            branch
            path
          }
          shipOpsRef {
            token
          }
        }
      }
    }
  }
`;
export const listWatches = gql(listWatchesRaw);

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
      allowRollback
      currentVersion {
        title
        status
        createdOn
        sequence
        releaseNotes
        deployedAt
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
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          deployedAt
          parentSequence
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          deployedAt
          parentSequence
        }
        gitops {
          enabled
          provider
          uri
          path
          branch
          format
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
          }
          gitOpsRef {
            owner
            repo
            branch
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

export const getKotsApplicationTreeRaw = `
  query getKotsApplicationTree($slug: String!, $sequence: Int!) {
    getKotsApplicationTree(slug: $slug, sequence: $sequence)
  }
`;

export const getKotsApplicationTree = gql(getKotsApplicationTreeRaw);

export const getKotsFilesRaw = `
  query getKotsFiles($slug: String!, $sequence: Int!, $fileNames: [String!]) {
    getKotsFiles(slug: $slug, sequence: $sequence, fileNames: $fileNames)
  }
`;

export const getKotsFiles = gql(getKotsFilesRaw);

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
      }
      gitOpsRef {
        owner
        repo
        branch
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
    }
  }
`;

export const getKotsDownstreamHistory = gql(getKotsDownstreamHistoryRaw);

export const getAppRegistryDetailsRaw = `
  query getAppRegistryDetails($slug: String!) {
    getAppRegistryDetails(slug: $slug) {
      registryHostname
      registryUsername
      registryPassword
      namespace
      lastSyncedAt
    }
  }
`;

export const getAppRegistryDetails = gql(getAppRegistryDetailsRaw);

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
        helpText
        recommended
        default
        value
        multiValue
        readOnly
        writeOnce
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
      entitlements {
        title
        value
        label
      }
    }
  }
`;

export const hasLicenseUpdates = gql`
  query hasLicenseUpdates($appSlug: String!) {
    hasLicenseUpdates(appSlug: $appSlug)
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
        helpText
        recommended
        default
        value
        multiValue
        readOnly
        writeOnce
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
`
