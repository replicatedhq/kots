import gql from "graphql-tag";

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
