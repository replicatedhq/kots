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
    }
  }
`;
export const listSupportBundles = gql(listSupportBundlesRaw);
