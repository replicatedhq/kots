import gql from "graphql-tag";

export const listClustersRaw = `
  query listClusters {
    listClusters {
      id
      title
      slug
      createdOn
      lastUpdated
      shipOpsRef {
        token
      }
      totalApplicationCount
    }
  }
`;
export const listClusters = gql(listClustersRaw);
