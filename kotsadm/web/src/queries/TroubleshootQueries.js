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

export const getSupportBundleRaw = `
  query getSupportBundle($watchSlug: String!) {
    getSupportBundle(watchSlug: $watchSlug) {
      id
      slug
      name
      size
      status
      treeIndex
      createdAt
      uploadedAt
      isArchived
      kotsLicenseType
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

export const getSupportBundle = gql(getSupportBundleRaw);
