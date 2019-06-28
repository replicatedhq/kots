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
