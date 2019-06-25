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
      }
      pendingVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
      }
      pastVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
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
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
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
      }
      pendingVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
      }
      pastVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
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
      }
      pendingVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
      }
      pastVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
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
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
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
      }
      pendingVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
      }
      pastVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
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
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
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
  query getParentWatch($id: String) {
    getParentWatch(id: $id) {
      watchName
      slug
      metadata
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
      }
      pendingVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
      }
      pastVersions {
        title
        status
        createdOn
        sequence
        pullrequestNumber
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
        }
        pendingVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
        }
        pastVersions {
          title
          status
          createdOn
          sequence
          pullrequestNumber
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

export const getWatchContributors = gql`
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
      clusterID
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
