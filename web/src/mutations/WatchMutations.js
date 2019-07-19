import gql from "graphql-tag";

export const checkForUpdatesRaw = `
  mutation checkForUpdates($watchId: ID!) {
    checkForUpdates(watchId: $watchId)
  }
`
export const checkForUpdates = gql(checkForUpdatesRaw);

export const createNewWatchRaw = `
  mutation createWatch($stateJSON: String!) {
    createWatch(stateJSON: $stateJSON) {
      id
      slug
      watchName
      createdOn
      lastUpdated
    }
  }
`;
export const createNewWatch = gql(createNewWatchRaw);

export const deleteWatchRaw = `
  mutation deleteWatch($watchId: String!, $childWatchIds: [String]) {
    deleteWatch(watchId: $watchId, childWatchIds: $childWatchIds)
  }
`;
export const deleteWatch = gql(deleteWatchRaw);

export const updateStateJSON = gql`
  mutation updateStateJSON($slug: String!, $stateJSON: String!) {
    updateStateJSON(slug: $slug, stateJSON: $stateJSON) {
      id
      slug
      stateJSON
    }
  }
`;

export const updateWatchRaw = `
  mutation updateWatch($watchId: String!, $watchName: String, $iconUri: String) {
    updateWatch(watchId: $watchId, watchName: $watchName, iconUri: $iconUri) {
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
    }
  }
`;
export const updateWatch = gql(updateWatchRaw);

export const deployWatchVersion = gql`
  mutation deployWatchVersion($watchId: String!, $sequence: Int) {
    deployWatchVersion(watchId: $watchId, sequence: $sequence)
  }
`;

export const createInitSessionRaw = `
  mutation createInitSession($pendingInitId: String, $upstreamUri: String, $clusterID: String, $githubPath: String) {
    createInitSession(pendingInitId: $pendingInitId, upstreamUri: $upstreamUri, clusterID: $clusterID, githubPath: $githubPath) {
      id
      upstreamUri
      createdOn
    }
  }
`
export const createInitSession = gql(createInitSessionRaw);

export const createUnforkSession = gql`
  mutation createUnforkSession($upstreamUri: String!, $forkUri: String!) {
    createUnforkSession(upstreamUri: $upstreamUri, forkUri: $forkUri) {
      id
      upstreamUri
      forkUri
      createdOn
    }
  }
`

export const createUpdateSession = gql`
  mutation createUpdateSession($watchId: ID!) {
    createUpdateSession(watchId: $watchId) {
      id
      watchId
      createdOn
    }
  }
`

export const createEditSessionRaw = `
  mutation createEditSession($watchId: ID!) {
    createEditSession(watchId: $watchId) {
      id
      watchId
      createdOn
    }
  }
`;
export const createEditSession = gql(createEditSessionRaw);

export const addWatchContributorRaw = `
  mutation addWatchContributor($watchId: ID!, $githubId: Int!, $login: String!, $avatarUrl: String) {
    addWatchContributor(watchId: $watchId, githubId: $githubId, login: $login, avatarUrl: $avatarUrl) {
      id
      createdAt
      githubId
      login
      avatar_url
    }
  }`;
export const addWatchContributor = gql(addWatchContributorRaw);

export const removeWatchContributorRaw = `
  mutation removeWatchContributor($watchId: ID!, $contributorId: String!) {
    removeWatchContributor(watchId: $watchId, contributorId: $contributorId) {
      id
      createdAt
      githubId
      login
      avatar_url
    }
  }`;
export const removeWatchContributor = gql(removeWatchContributorRaw);

export const deleteNotification = gql`
  mutation deleteNotification ($id: String!, $isPending: Boolean) {
    deleteNotification(id: $id, isPending: $isPending)
  }
`;

export const syncWatchLicense = gql`
  mutation syncWatchLicense ($watchId: String!, $licenseId: String!, $entitlementSpec: String) {
    syncWatchLicense(watchId: $watchId, licenseId: $licenseId, entitlementSpec: $entitlementSpec) {
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
