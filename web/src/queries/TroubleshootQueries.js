import gql from "graphql-tag";

export const listSupportBundlesRaw = `
  query listSupportBundles($watchSlug: String!) {
    listSupportBundles(watchSlug: $watchSlug) {
      id
      slug
      name
      size
      status
      treeIndex
      createdAt
      uploadedAt
      isArchived
      analysis {
        id
        error
        maxSeverity
        createdAt
        insights {
          key
          severity
          primary
          detail
          icon
          icon_key
          desiredPosition
        }
      }
    }
  }
`;
export const listSupportBundles = gql(listSupportBundlesRaw);
export const getAnalysisInsights = gql`
  query getAnalysisInsights($slug: String!) {
    getAnalysisInsights(slug: $slug) {
      bundle {
        id,
        size,
        name,
        teamId,
        teamName,
        teamShareIds,
        status,
        createdAt,
        viewed,
        slug,
        customer {
          id,
          name,
          avatar
        },
        uri,
        signedUri,
        notes,
        treeIndex,
      },
      insights {
        level
        primary
        key
        detail
        icon
        icon_key
        desiredPosition
        labels {
          key,
          value
        }
      }
    }
  }
`;

export const analysisFiles = gql`
  query analysisFiles($bundleId: ID!, $fileNames: [String!]) {
    analysisFiles(bundleId: $bundleId, fileNames: $fileNames)
  }
`;

export const getGenerateBundleCommand = gql`
query getGenerateBundleCommand($customerId: ID, $channelId: ID) {
  getGenerateBundleCommand(customerId: $customerId, channelId: $channelId)
}
`;
