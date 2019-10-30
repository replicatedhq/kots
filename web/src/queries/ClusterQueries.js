import gql from "graphql-tag";

export const listClustersRaw = `
  query listClusters {
    listClusters {
      id
      title
      slug
      createdOn
      lastUpdated
      isGitOps
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
export const listClusters = gql(listClustersRaw);
