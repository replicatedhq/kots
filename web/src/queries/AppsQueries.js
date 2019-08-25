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
        name
        iconUri
        createdAt
        updatedAt
        slug
        currentSequence
        lastUpdateCheckAt
        downstreams {
          name
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
      currentSequence
      lastUpdateCheckAt
      downstreams {
        name
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
