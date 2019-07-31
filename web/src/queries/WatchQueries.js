import gql from "graphql-tag";

export const getWatchVersionRaw = `
  query getWatchVersion($id: String!, $sequence: Int) {
    getWatchVersion(id: $id, sequence: $sequence) {
      title
      status
      createdOn
      sequence
      pullrequestNumber
      rendered
    }
  }
`;
export const getWatchVersion = gql(getWatchVersionRaw);

export const listWatchesRaw = `
  query listWatches {
    listWatches {
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

export const searchWatches = gql`
  query searchWatches($watchName: String!) {
    searchWatches(watchName: $watchName) {
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
    }
  }
`;

export const getWatch = gql`
  query getWatch($slug: String) {
    getWatch(slug: $slug) {
      id
      stateJSON
      watchName
      slug
      watchIcon
      createdOn
      lastUpdated
      metadata
      entitlements {
        key
        value
        name
      }
      config {
        name
        title
        description
        items {
          name
          title
          default
          type
          value
        }
      }
      lastUpdateCheck
      bundleCommand
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
      watches {
        id
        stateJSON
        watchName
        slug
        watchIcon
        createdOn
        lastUpdated
        lastUpdateCheck
        bundleCommand
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

export const getWatchJson = gql`
  query getWatch($slug: String) {
    getWatch(slug: $slug) {
      id
      slug
      stateJSON
    }
  }
`;

export const getWatchById = gql`
  query getWatch($id: String) {
    getWatch(id: $id) {
      watchName
      slug
      metadata
      lastUpdateCheck
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

export const getParentWatchRaw = `
  query getParentWatch($id: String, $slug: String) {
    getParentWatch(id: $id, slug: $slug) {
      watchName
      slug
      metadata
      lastUpdateCheck
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

export const getParentWatch = gql(getParentWatchRaw);

export const listNotificationsQuery = gql`
  query listNotifications($watchId: String!) {
    listNotifications(watchId: $watchId) {
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
  }
`;

export const getNotification = gql`
  query getNotification($notificationId: String!) {
    getNotification(notificationId: $notificationId) {
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
  }
`;

export const getWatchContributorsRaw = `
  query watchContributors($id: String!) {
    watchContributors(id: $id) {
      id
      createdAt
      githubId
      login
      avatar_url
    }
  }
`;

export const getWatchContributors = gql(getWatchContributorsRaw);

export const pullRequestHistory = gql`
  query pullRequestHistory($notificationId: String!) {
    pullRequestHistory(notificationId: $notificationId) {
      title
      status
      createdOn
      number
      uri
      sequence
    }
  }
`;

export const userFeatures = gql`
  query userFeatures {
    userFeatures {
      id
    }
  }
`;

export const listPendingInitRaw = `
  query listPendingInitSessions {
    listPendingInitSessions {
      id
      title
      upstreamURI
    }
  }
`;

export const listPendingInit = gql(listPendingInitRaw);

export const listHelmChartsRaw = `
  query listHelmCharts {
    listHelmCharts {
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
  }
`;

export const listHelmCharts = gql(listHelmChartsRaw);

export const searchPendingInitSessionsRaw = `
  query searchPendingInitSessions($title: String!) {
    searchPendingInitSessions(title: $title) {
      id
      title
      upstreamURI
    }
  }
`;

export const searchPendingInitSessions = gql(searchPendingInitSessionsRaw);

export const getHelmChartRaw = `
  query getHelmChart($id: String!) {
      getHelmChart(id: $id) {
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
  }
`

export const getHelmChart = gql(getHelmChartRaw);

export const getDownstreamHistoryRaw = `
  query getDownstreamHistory($slug: String!) {
    getDownstreamHistory(slug: $slug) {
      title
      status
      createdOn
      sequence
      pullrequestNumber
    }
  }
`;

export const getWatchLicense = gql`
  query getWatchLicense($watchId: String!) {
    getWatchLicense(watchId: $watchId) {
      id
      channel
      createdAt
      expiresAt
      type
      entitlements {
        key
        value
        name
      }
    }
  }
`;

export const getLatestWatchLicense = gql`
  query getLatestWatchLicense($licenseId: String!) {
    getLatestWatchLicense(licenseId: $licenseId) {
      id
      channel
      createdAt
      expiresAt
      type
      entitlements {
        key
        value
        name
      }
    }
  }
`;

export const getDownstreamHistory = gql(getDownstreamHistoryRaw);

export const getApplicationTreeRaw = `
  query getApplicationTree($slug: String!, $sequence: Int!) {
    getApplicationTree(slug: $slug, sequence: $sequence)
  }
`;

export const getApplicationTree = gql(getApplicationTreeRaw);

export const getFilesRaw = `
  query getFiles($slug: String!, $sequence: Int!, $fileNames: [String!]) {
    getFiles(slug: $slug, sequence: $sequence, fileNames: $fileNames)
  }
`;

export const getFiles = gql(getFilesRaw)
